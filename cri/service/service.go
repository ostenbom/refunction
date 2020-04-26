package service

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/containerd/containerd"
	containerdCRIconfig "github.com/containerd/cri/pkg/config"
	containerdCRIserver "github.com/containerd/cri/pkg/server"
	"github.com/ostenbom/refunction/controller"
	"google.golang.org/grpc"
	runtime "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

const (
	kubeAPIVersion            = "0.1.0"
	runtimeAPIVersion         = "v1alpha2"
	defaultContainerdRootDir  = "/var/lib/containerd"
	defaultContainerdStateDir = "/run/containerd"
)

type CRIService interface {
	runtime.RuntimeServiceServer
	runtime.ImageServiceServer
	Register(*grpc.Server)
	GetController(string) (controller.Controller, error)
}

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . ContainerdCRIService

type ContainerdCRIService interface {
	containerdCRIserver.CRIService
}

type criService struct {
	client        *containerd.Client
	containerdCRI ContainerdCRIService
	controllers   map[string]controller.Controller
}

type ContainerInfo struct {
	Pid int `json:"pid"`
}

func NewCRIService(client *containerd.Client) (CRIService, error) {
	containerdPluginConf := containerdCRIconfig.DefaultConfig()
	containerdPluginConf.ContainerdConfig.Runtimes["runc"] = containerdCRIconfig.Runtime{
		Type: "io.containerd.runtime.v1.linux",
	}

	criConfig := containerdCRIconfig.Config{
		PluginConfig:       containerdPluginConf,
		ContainerdRootDir:  defaultContainerdRootDir,
		ContainerdEndpoint: filepath.Join(defaultContainerdStateDir, "containerd.sock"),
		RootDir:            filepath.Join(defaultContainerdRootDir, "refunction.v1.cri"),
		StateDir:           defaultContainerdStateDir,
	}

	containerdCRI, err := containerdCRIserver.NewCRIService(criConfig, client)
	if err != nil {
		return nil, fmt.Errorf("could not start containerdCRI: %v", err)
	}

	go func() {
		if err := containerdCRI.Run(); err != nil {
			log.Fatalf("containerdCRI run error: %v\n", err)
		}
	}()

	c := &criService{
		client:        client,
		containerdCRI: containerdCRI,
		controllers:   make(map[string]controller.Controller),
	}

	return c, nil
}

func NewFakeCRIService(containerdCRI ContainerdCRIService) CRIService {
	c := &criService{
		client:        &containerd.Client{},
		containerdCRI: containerdCRI,
		controllers:   make(map[string]controller.Controller),
	}

	return c
}

func (c *criService) Register(s *grpc.Server) {
	runtime.RegisterRuntimeServiceServer(s, c)
	runtime.RegisterImageServiceServer(s, c)
}

func (c *criService) GetController(id string) (controller.Controller, error) {
	controller, exists := c.controllers[id]

	if !exists {
		return nil, fmt.Errorf("no such controller for id: %s", id)
	}

	return controller, nil
}

// Version returns the runtime name, runtime version, and runtime API version.
func (c *criService) Version(context.Context, *runtime.VersionRequest) (*runtime.VersionResponse, error) {
	return &runtime.VersionResponse{
		Version:           kubeAPIVersion,
		RuntimeName:       "refunction",
		RuntimeApiVersion: runtimeAPIVersion,
		RuntimeVersion:    "0.0.1",
	}, nil
}

// RunPodSandbox creates and starts a pod-level sandbox. Runtimes must ensure
// the sandbox is in the ready state on success.
func (c *criService) RunPodSandbox(ctx context.Context, req *runtime.RunPodSandboxRequest) (*runtime.RunPodSandboxResponse, error) {
	return c.containerdCRI.RunPodSandbox(ctx, req)
}

// StopPodSandbox stops any running process that is part of the sandbox and
// reclaims network resources (e.g., IP addresses) allocated to the sandbox.
// If there are any running containers in the sandbox, they must be forcibly
// terminated.
// This call is idempotent, and must not return an error if all relevant
// resources have already been reclaimed. kubelet will call StopPodSandbox
// at least once before calling RemovePodSandbox. It will also attempt to
// reclaim resources eagerly, as soon as a sandbox is not needed. Hence,
// multiple StopPodSandbox calls are expected.
func (c *criService) StopPodSandbox(ctx context.Context, req *runtime.StopPodSandboxRequest) (*runtime.StopPodSandboxResponse, error) {
	return c.containerdCRI.StopPodSandbox(ctx, req)
}

// RemovePodSandbox removes the sandbox. If there are any running containers
// in the sandbox, they must be forcibly terminated and removed.
// This call is idempotent, and must not return an error if the sandbox has
// already been removed.
func (c *criService) RemovePodSandbox(ctx context.Context, req *runtime.RemovePodSandboxRequest) (*runtime.RemovePodSandboxResponse, error) {
	return c.containerdCRI.RemovePodSandbox(ctx, req)
}

// PodSandboxStatus returns the status of the PodSandbox. If the PodSandbox is not
// present, returns an error.
func (c *criService) PodSandboxStatus(ctx context.Context, req *runtime.PodSandboxStatusRequest) (*runtime.PodSandboxStatusResponse, error) {
	return c.containerdCRI.PodSandboxStatus(ctx, req)
}

// ListPodSandbox returns a list of PodSandboxes.
func (c *criService) ListPodSandbox(ctx context.Context, req *runtime.ListPodSandboxRequest) (*runtime.ListPodSandboxResponse, error) {
	return c.containerdCRI.ListPodSandbox(ctx, req)
}

// CreateContainer creates a new container in specified PodSandbox
func (c *criService) CreateContainer(ctx context.Context, req *runtime.CreateContainerRequest) (*runtime.CreateContainerResponse, error) {
	createResponse, err := c.containerdCRI.CreateContainer(ctx, req)
	if err != nil {
		return nil, err
	}

	_, isRefunctionPod := req.GetSandboxConfig().GetAnnotations()["refunction"]
	if isRefunctionPod {
		// TODO: Probably dissallow the tty case
		if req.GetConfig().GetTty() {
			fmt.Println("Unhandled: TTY refunction container!")
		}

		c.controllers[createResponse.GetContainerId()] = controller.NewController()
	}

	return createResponse, nil
}

// StopContainer stops a running container with a grace period (i.e., timeout).
// This call is idempotent, and must not return an error if the container has
// already been stopped.
// TODO: what must the runtime do after the grace period is reached?
func (c *criService) StopContainer(ctx context.Context, req *runtime.StopContainerRequest) (*runtime.StopContainerResponse, error) {
	return c.containerdCRI.StopContainer(ctx, req)
}

// RemoveContainer removes the container. If the container is running, the
// container must be forcibly removed.
// This call is idempotent, and must not return an error if the container has
// already been removed.
func (c *criService) RemoveContainer(ctx context.Context, req *runtime.RemoveContainerRequest) (*runtime.RemoveContainerResponse, error) {
	return c.containerdCRI.RemoveContainer(ctx, req)
}

// ListContainers lists all containers by filters.
func (c *criService) ListContainers(ctx context.Context, req *runtime.ListContainersRequest) (*runtime.ListContainersResponse, error) {
	return c.containerdCRI.ListContainers(ctx, req)
}

// ContainerStatus returns status of the container. If the container is not
// present, returns an error.
func (c *criService) ContainerStatus(ctx context.Context, req *runtime.ContainerStatusRequest) (*runtime.ContainerStatusResponse, error) {
	return c.containerdCRI.ContainerStatus(ctx, req)
}

// UpdateContainerResources updates ContainerConfig of the container.
func (c *criService) UpdateContainerResources(ctx context.Context, req *runtime.UpdateContainerResourcesRequest) (*runtime.UpdateContainerResourcesResponse, error) {
	return c.containerdCRI.UpdateContainerResources(ctx, req)
}

// ReopenContainerLog asks runtime to reopen the stdout/stderr log file
// for the container. This is often called after the log file has been
// rotated. If the container is not running, container runtime can choose
// to either create a new log file and return &runtime.{}, or return an error.
// Once it returns error, new container log file MUST NOT be created.
func (c *criService) ReopenContainerLog(ctx context.Context, req *runtime.ReopenContainerLogRequest) (*runtime.ReopenContainerLogResponse, error) {
	return c.containerdCRI.ReopenContainerLog(ctx, req)
}

// ExecSync runs a command in a container synchronously.
func (c *criService) ExecSync(ctx context.Context, req *runtime.ExecSyncRequest) (*runtime.ExecSyncResponse, error) {
	return c.containerdCRI.ExecSync(ctx, req)
}

// Exec prepares a streaming endpoint to execute a command in the container.
func (c *criService) Exec(ctx context.Context, req *runtime.ExecRequest) (*runtime.ExecResponse, error) {
	return c.containerdCRI.Exec(ctx, req)
}

// Attach prepares a streaming endpoint to attach to a running container.
func (c *criService) Attach(ctx context.Context, req *runtime.AttachRequest) (*runtime.AttachResponse, error) {
	return c.containerdCRI.Attach(ctx, req)
}

// PortForward prepares a streaming endpoint to forward ports from a PodSandbox.
func (c *criService) PortForward(ctx context.Context, req *runtime.PortForwardRequest) (*runtime.PortForwardResponse, error) {
	return c.containerdCRI.PortForward(ctx, req)
}

// ContainerStats returns stats of the container. If the container does not
// exist, the call returns an error.
func (c *criService) ContainerStats(ctx context.Context, req *runtime.ContainerStatsRequest) (*runtime.ContainerStatsResponse, error) {
	return c.containerdCRI.ContainerStats(ctx, req)
}

// ListContainerStats returns stats of all running containers.
func (c *criService) ListContainerStats(ctx context.Context, req *runtime.ListContainerStatsRequest) (*runtime.ListContainerStatsResponse, error) {
	return c.containerdCRI.ListContainerStats(ctx, req)
}

// UpdateRuntimeConfig updates the runtime configuration based on the given request.
func (c *criService) UpdateRuntimeConfig(ctx context.Context, req *runtime.UpdateRuntimeConfigRequest) (*runtime.UpdateRuntimeConfigResponse, error) {
	return c.containerdCRI.UpdateRuntimeConfig(ctx, req)
}

// Status returns the status of the runtime.
func (c *criService) Status(ctx context.Context, req *runtime.StatusRequest) (*runtime.StatusResponse, error) {
	return c.containerdCRI.Status(ctx, req)
}

// ListImages lists existing images.
func (c *criService) ListImages(ctx context.Context, req *runtime.ListImagesRequest) (*runtime.ListImagesResponse, error) {
	return c.containerdCRI.ListImages(ctx, req)
}

// ImageStatus returns the status of the image. If the image is not
// present, returns a response with ImageStatusResponse.Image set to
// nil.
func (c *criService) ImageStatus(ctx context.Context, req *runtime.ImageStatusRequest) (*runtime.ImageStatusResponse, error) {
	return c.containerdCRI.ImageStatus(ctx, req)
}

// PullImage pulls an image with authentication config.
func (c *criService) PullImage(ctx context.Context, req *runtime.PullImageRequest) (*runtime.PullImageResponse, error) {
	return c.containerdCRI.PullImage(ctx, req)
}

// RemoveImage removes the image.
// This call is idempotent, and must not return an error if the image has
// already been removed.
func (c *criService) RemoveImage(ctx context.Context, req *runtime.RemoveImageRequest) (*runtime.RemoveImageResponse, error) {
	return c.containerdCRI.RemoveImage(ctx, req)
}

// ImageFSInfo returns information of the filesystem that is used to store images.
func (c *criService) ImageFsInfo(ctx context.Context, req *runtime.ImageFsInfoRequest) (*runtime.ImageFsInfoResponse, error) {
	return c.containerdCRI.ImageFsInfo(ctx, req)
}

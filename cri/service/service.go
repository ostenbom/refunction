package service

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/containerd/containerd"
	containerdCRIconfig "github.com/containerd/cri/pkg/config"
	containerdCRIserver "github.com/containerd/cri/pkg/server"
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
	register(*grpc.Server)
}

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . ContainerdCRIService
type ContainerdCRIService interface {
	containerdCRIserver.CRIService
}

type criService struct {
	client        *containerd.Client
	containerdCRI ContainerdCRIService
}

func NewCRIService(client *containerd.Client) (CRIService, error) {
	criConfig := containerdCRIconfig.Config{
		PluginConfig:       containerdCRIconfig.DefaultConfig(),
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
	}

	return c, nil
}

func NewFakeCRIService(containerdCRI ContainerdCRIService) CRIService {
	c := &criService{
		client:        &containerd.Client{},
		containerdCRI: containerdCRI,
	}

	return c
}

func (c *criService) register(s *grpc.Server) {
	runtime.RegisterRuntimeServiceServer(s, c)
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
func (c *criService) RunPodSandbox(context.Context, *runtime.RunPodSandboxRequest) (*runtime.RunPodSandboxResponse, error) {
	return &runtime.RunPodSandboxResponse{}, nil
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
func (c *criService) StopPodSandbox(context.Context, *runtime.StopPodSandboxRequest) (*runtime.StopPodSandboxResponse, error) {
	return &runtime.StopPodSandboxResponse{}, nil
}

// RemovePodSandbox removes the sandbox. If there are any running containers
// in the sandbox, they must be forcibly terminated and removed.
// This call is idempotent, and must not return an error if the sandbox has
// already been removed.
func (c *criService) RemovePodSandbox(context.Context, *runtime.RemovePodSandboxRequest) (*runtime.RemovePodSandboxResponse, error) {
	return &runtime.RemovePodSandboxResponse{}, nil
}

// PodSandboxStatus returns the status of the PodSandbox. If the PodSandbox is not
// present, returns an error.
func (c *criService) PodSandboxStatus(context.Context, *runtime.PodSandboxStatusRequest) (*runtime.PodSandboxStatusResponse, error) {
	return &runtime.PodSandboxStatusResponse{}, nil
}

// ListPodSandbox returns a list of PodSandboxes.
func (c *criService) ListPodSandbox(context.Context, *runtime.ListPodSandboxRequest) (*runtime.ListPodSandboxResponse, error) {
	return &runtime.ListPodSandboxResponse{}, nil
}

// CreateContainer creates a new container in specified PodSandbox
func (c *criService) CreateContainer(context.Context, *runtime.CreateContainerRequest) (*runtime.CreateContainerResponse, error) {
	return &runtime.CreateContainerResponse{}, nil
}

// StartContainer starts the container.
func (c *criService) StartContainer(context.Context, *runtime.StartContainerRequest) (*runtime.StartContainerResponse, error) {
	return &runtime.StartContainerResponse{}, nil
}

// StopContainer stops a running container with a grace period (i.e., timeout).
// This call is idempotent, and must not return an error if the container has
// already been stopped.
// TODO: what must the runtime do after the grace period is reached?
func (c *criService) StopContainer(context.Context, *runtime.StopContainerRequest) (*runtime.StopContainerResponse, error) {
	return &runtime.StopContainerResponse{}, nil
}

// RemoveContainer removes the container. If the container is running, the
// container must be forcibly removed.
// This call is idempotent, and must not return an error if the container has
// already been removed.
func (c *criService) RemoveContainer(context.Context, *runtime.RemoveContainerRequest) (*runtime.RemoveContainerResponse, error) {
	return &runtime.RemoveContainerResponse{}, nil
}

// ListContainers lists all containers by filters.
func (c *criService) ListContainers(context.Context, *runtime.ListContainersRequest) (*runtime.ListContainersResponse, error) {
	return &runtime.ListContainersResponse{}, nil
}

// ContainerStatus returns status of the container. If the container is not
// present, returns an error.
func (c *criService) ContainerStatus(context.Context, *runtime.ContainerStatusRequest) (*runtime.ContainerStatusResponse, error) {
	return &runtime.ContainerStatusResponse{}, nil
}

// UpdateContainerResources updates ContainerConfig of the container.
func (c *criService) UpdateContainerResources(context.Context, *runtime.UpdateContainerResourcesRequest) (*runtime.UpdateContainerResourcesResponse, error) {
	return &runtime.UpdateContainerResourcesResponse{}, nil
}

// ReopenContainerLog asks runtime to reopen the stdout/stderr log file
// for the container. This is often called after the log file has been
// rotated. If the container is not running, container runtime can choose
// to either create a new log file and return &runtime.{}, or return an error.
// Once it returns error, new container log file MUST NOT be created.
func (c *criService) ReopenContainerLog(context.Context, *runtime.ReopenContainerLogRequest) (*runtime.ReopenContainerLogResponse, error) {
	return &runtime.ReopenContainerLogResponse{}, nil
}

// ExecSync runs a command in a container synchronously.
func (c *criService) ExecSync(context.Context, *runtime.ExecSyncRequest) (*runtime.ExecSyncResponse, error) {
	return &runtime.ExecSyncResponse{}, nil
}

// Exec prepares a streaming endpoint to execute a command in the container.
func (c *criService) Exec(context.Context, *runtime.ExecRequest) (*runtime.ExecResponse, error) {
	return &runtime.ExecResponse{}, nil
}

// Attach prepares a streaming endpoint to attach to a running container.
func (c *criService) Attach(context.Context, *runtime.AttachRequest) (*runtime.AttachResponse, error) {
	return &runtime.AttachResponse{}, nil
}

// PortForward prepares a streaming endpoint to forward ports from a PodSandbox.
func (c *criService) PortForward(context.Context, *runtime.PortForwardRequest) (*runtime.PortForwardResponse, error) {
	return &runtime.PortForwardResponse{}, nil
}

// ContainerStats returns stats of the container. If the container does not
// exist, the call returns an error.
func (c *criService) ContainerStats(context.Context, *runtime.ContainerStatsRequest) (*runtime.ContainerStatsResponse, error) {
	return &runtime.ContainerStatsResponse{}, nil
}

// ListContainerStats returns stats of all running containers.
func (c *criService) ListContainerStats(context.Context, *runtime.ListContainerStatsRequest) (*runtime.ListContainerStatsResponse, error) {
	return &runtime.ListContainerStatsResponse{}, nil
}

// UpdateRuntimeConfig updates the runtime configuration based on the given request.
func (c *criService) UpdateRuntimeConfig(context.Context, *runtime.UpdateRuntimeConfigRequest) (*runtime.UpdateRuntimeConfigResponse, error) {
	return &runtime.UpdateRuntimeConfigResponse{}, nil
}

// Status returns the status of the runtime.
func (c *criService) Status(context.Context, *runtime.StatusRequest) (*runtime.StatusResponse, error) {
	return &runtime.StatusResponse{}, nil
}

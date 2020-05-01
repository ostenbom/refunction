package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"github.com/ostenbom/refunction/controller"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	runtime "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

// StartContainer starts the container.
func (c *criService) StartContainer(ctx context.Context, req *runtime.StartContainerRequest) (*runtime.StartContainerResponse, error) {
	startResp, err := c.containerdCRI.StartContainer(ctx, req)
	if err != nil {
		return nil, err
	}

	control, isRefunctionContainer := c.controllers.Controller(req.GetContainerId())
	if !isRefunctionContainer {
		return startResp, nil
	}

	containerId := req.GetContainerId()

	err = c.setControllerPid(ctx, containerId, control)
	if err != nil {
		return nil, fmt.Errorf("could not set started container pid: %s", err)
	}

	err = c.setControllerStreams(ctx, containerId, control)
	if err != nil {
		return nil, fmt.Errorf("could not set started container streams: %s", err)
	}

	return startResp, nil
}

func (c *criService) setControllerPid(ctx context.Context, containerId string, control controller.Controller) error {

	statusReq := &runtime.ContainerStatusRequest{
		ContainerId: containerId,
		Verbose:     true,
	}

	statusResp, err := c.containerdCRI.ContainerStatus(ctx, statusReq)
	if err != nil {
		return fmt.Errorf("started container status error: %s", err)
	}

	var info ContainerInfo

	infoString := statusResp.GetInfo()["info"]

	err = json.Unmarshal([]byte(infoString), &info)
	if err != nil {
		return fmt.Errorf("could not parse container info: %s", err)
	}

	control.SetPid(info.Pid)

	return nil
}

func (c *criService) setControllerStreams(ctx context.Context, containerId string, control controller.Controller) error {
	attachReq := &runtime.AttachRequest{
		ContainerId: containerId,
		Stdin:       true,
		Stdout:      true,
		Stderr:      true,
	}

	attachResp, err := c.containerdCRI.Attach(ctx, attachReq)
	if err != nil {
		return fmt.Errorf("could not attach to refunction container: %s", err)
	}

	fmt.Printf("Pretty amazing attach response: %s\n", attachResp.GetUrl())
	attachUrl, err := url.Parse(attachResp.GetUrl())
	if err != nil {
		fmt.Printf("found it hard to parse that: %s\n", err)
		return err
	}
	fmt.Printf("Pretty amazing parsed attach url: %s\n", attachUrl)

	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()

	control.SetStreams(stdinW, stdoutR, stderrR)

	streamErr := make(chan error)
	go func() {
		exec, err := remotecommand.NewSPDYExecutor(&restclient.Config{}, "POST", attachUrl)
		streamErr <- err

		opts := remotecommand.StreamOptions{
			Stdin:  stdinR,
			Stdout: stdoutW,
			Stderr: stderrW,
			Tty:    false,
		}

		err = exec.Stream(opts)
		if err != nil {
			// TODO: a channel to handle these errors in the controller
			fmt.Printf("stream error for container %s: %s\n", containerId, err)
		}
	}()

	err = <-streamErr
	if err != nil {
		return fmt.Errorf("could not stream refunction container: %s", err)
	}

	return nil
}

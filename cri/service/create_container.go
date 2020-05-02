package service

import (
	"context"
	"fmt"

	runtime "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

// CreateContainer creates a new container in specified PodSandbox
func (c *criService) CreateContainer(ctx context.Context, req *runtime.CreateContainerRequest) (*runtime.CreateContainerResponse, error) {
	_, isRefunctionPod := req.GetSandboxConfig().GetAnnotations()["refunction"]
	if isRefunctionPod {
		req.GetConfig().Stdin = true
	}

	createResponse, err := c.containerdCRI.CreateContainer(ctx, req)
	if err != nil {
		return nil, err
	}

	if isRefunctionPod {
		// TODO: Probably dissallow the tty case
		if req.GetConfig().GetTty() {
			fmt.Println("Unhandled: TTY refunction container!")
		}

		c.controllers.CreateController(createResponse.GetContainerId())
	}

	return createResponse, nil
}

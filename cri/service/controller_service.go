package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ostenbom/refunction/controller"
	refunction "github.com/ostenbom/refunction/cri/service/api/refunction/v1alpha"
)

type ControllerService interface {
	Controller(string) (controller.Controller, bool)
	CreateController(string)
	refunction.RefunctionServiceServer
	// Register(s *grpc.Server)
}

type controllerService struct {
	controllers        map[string]controller.Controller
	controller_creator func() controller.Controller
}

func NewControllerService() ControllerService {
	return &controllerService{
		controllers:        make(map[string]controller.Controller),
		controller_creator: func() controller.Controller { return controller.NewController() },
	}
}

func NewFakeControllerService(creator func() controller.Controller) ControllerService {
	return &controllerService{
		controllers:        make(map[string]controller.Controller),
		controller_creator: creator,
	}
}

func (s *controllerService) Controller(id string) (controller.Controller, bool) {
	c, exists := s.controllers[id]
	return c, exists
}

func (s *controllerService) CreateController(id string) {
	s.controllers[id] = s.controller_creator()
}

func (s *controllerService) ListContainers(ctx context.Context, req *refunction.ListContainersRequest) (*refunction.ListContainersResponse, error) {
	ids := make([]string, len(s.controllers))

	i := 0
	for k := range s.controllers {
		ids[i] = k
		i++
	}

	return &refunction.ListContainersResponse{
		ContainerIds: ids,
	}, nil
}

func (s *controllerService) SendRequest(ctx context.Context, req *refunction.Request) (*refunction.Response, error) {
	controller, exists := s.controllers[req.ContainerId]
	if !exists {
		return nil, fmt.Errorf("no such controller: %s", req.ContainerId)
	}

	var request interface{}

	err := json.Unmarshal([]byte(req.Request), &request)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal json request %s: %s", req.Request, err)
	}

	response, err := controller.SendRequest(request)
	if err != nil {
		return nil, fmt.Errorf("error sending request to container %s: %s", req.ContainerId, err)
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("could not marshal response to bytes %v+: %s", response, err)
	}

	return &refunction.Response{
		Response: string(responseBytes),
	}, nil
}

func (s *controllerService) SendFunction(ctx context.Context, req *refunction.FunctionRequest) (*refunction.FunctionResponse, error) {
	controller, exists := s.controllers[req.ContainerId]
	if !exists {
		return nil, fmt.Errorf("no such controller: %s", req.ContainerId)
	}

	err := controller.SendFunction(req.Function)
	if err != nil {
		return nil, fmt.Errorf("error sending function to container %s: %s", req.ContainerId, err)
	}
	return &refunction.FunctionResponse{}, nil
}

func (s *controllerService) Restore(ctx context.Context, req *refunction.RestoreRequest) (*refunction.RestoreResponse, error) {
	return &refunction.RestoreResponse{}, nil
}

package service

import (
	"context"

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
	controllers map[string]controller.Controller
}

func NewControllerService() ControllerService {
	return &controllerService{
		controllers: make(map[string]controller.Controller),
	}
}

func (s *controllerService) Controller(id string) (controller.Controller, bool) {
	c, exists := s.controllers[id]
	return c, exists
}

func (s *controllerService) CreateController(id string) {
	s.controllers[id] = controller.NewController()
}

func (s *controllerService) ListContainers(ctx context.Context, req *refunction.ListContainersRequest) (*refunction.ListContainersResponse, error) {
	return &refunction.ListContainersResponse{}, nil
}

func (s *controllerService) SendRequest(ctx context.Context, req *refunction.Request) (*refunction.Response, error) {
	return &refunction.Response{}, nil
}

func (s *controllerService) SendFunction(ctx context.Context, req *refunction.FunctionRequest) (*refunction.FunctionResponse, error) {
	return &refunction.FunctionResponse{}, nil
}

func (s *controllerService) Restore(ctx context.Context, req *refunction.RestoreRequest) (*refunction.RestoreResponse, error) {
	return &refunction.RestoreResponse{}, nil
}

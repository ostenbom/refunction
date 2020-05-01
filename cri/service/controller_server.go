package service

import (
	"github.com/ostenbom/refunction/controller"
)

type ControllerServer interface {
	Start()
	Controller(string) (controller.Controller, bool)
	CreateController(string)
}

type controllerServer struct {
	controllers map[string]controller.Controller
}

func NewControllerServer() ControllerServer {
	return &controllerServer{
		controllers: make(map[string]controller.Controller),
	}
}

func (s *controllerServer) Start() {

}

func (s *controllerServer) Controller(id string) (controller.Controller, bool) {
	c, exists := s.controllers[id]
	return c, exists
}

func (s *controllerServer) CreateController(id string) {
	s.controllers[id] = controller.NewController()
}

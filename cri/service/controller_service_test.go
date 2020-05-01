package service_test

import (
	"context"
	"sort"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/ostenbom/refunction/controller"
	"github.com/ostenbom/refunction/controller/controllerfakes"
	. "github.com/ostenbom/refunction/cri/service"
	refunction "github.com/ostenbom/refunction/cri/service/api/refunction/v1alpha"
)

var _ = Describe("ControllerService", func() {
	var service ControllerService
	ctx := context.Background()
	var createdControllers []*controllerfakes.FakeController

	fake_creator := func() controller.Controller {
		c := new(controllerfakes.FakeController)
		createdControllers = append(createdControllers, c)
		return c
	}

	BeforeEach(func() {
		createdControllers = []*controllerfakes.FakeController{}
		service = NewFakeControllerService(fake_creator)
	})

	Context("ListContainers", func() {
		It("returns an empty list when no containers created", func() {
			resp, err := service.ListContainers(ctx, &refunction.ListContainersRequest{})
			Expect(err).To(BeNil())
			Expect(len(resp.ContainerIds)).To(Equal(0))
		})

		It("returns correct containerIds when controllers have been created", func() {
			service.CreateController("first")
			service.CreateController("second")

			resp, err := service.ListContainers(ctx, &refunction.ListContainersRequest{})
			Expect(err).To(BeNil())

			sort.Strings(resp.ContainerIds)
			Expect(len(resp.ContainerIds)).To(Equal(2))
			Expect(resp.ContainerIds).To(Equal([]string{"first", "second"}))
		})
	})

	Context("SendFunction", func() {
		BeforeEach(func() {
			service.CreateController("first")
			service.CreateController("second")
		})

		It("sends the function to the correct controller", func() {
			function := "function: 1 + 1"
			_, err := service.SendFunction(ctx, &refunction.FunctionRequest{
				ContainerId: "second",
				Function:    function,
			})
			Expect(err).To(BeNil())

			Expect(createdControllers[1].SendFunctionCallCount()).To(Equal(1))
			calledFunction := createdControllers[1].SendFunctionArgsForCall(0)
			Expect(calledFunction).To(Equal(function))
		})
	})

	Context("SendRequest", func() {
		BeforeEach(func() {
			service.CreateController("first")
			service.CreateController("second")
		})

		It("sends the request to the correct controller", func() {
			createdControllers[1].SendRequestReturns(map[string]interface{}{"back": "grape"}, nil)

			request := "{\"name\": \"potato\"}"
			response, err := service.SendRequest(ctx, &refunction.Request{
				ContainerId: "second",
				Request:     request,
			})
			Expect(err).To(BeNil())

			Expect(createdControllers[1].SendRequestCallCount()).To(Equal(1))
			calledRequest := createdControllers[1].SendRequestArgsForCall(0)
			Expect(calledRequest).To(Equal(map[string]interface{}{"name": "potato"}))

			Expect(response.Response).To(Equal("{\"back\":\"grape\"}"))
		})
	})
})

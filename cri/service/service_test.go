package service_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"context"

	. "github.com/ostenbom/refunction/cri/service"
	"github.com/ostenbom/refunction/cri/service/servicefakes"
)

var _ = Describe("CRI Service", func() {
	var c CRIService
	ctx := context.Background()

	BeforeEach(func() {
		fakeContainerdCRI := new(servicefakes.FakeContainerdCRIService)
		c = NewFakeCRIService(fakeContainerdCRI)
	})

	Describe("Version", func() {
		It("returns the RuntimeName", func() {
			version, err := c.Version(ctx, nil)
			Expect(err).To(BeNil())
			Expect(version.RuntimeName).To(Equal("refunction"))
		})

		It("returns the 'Version' = kubeAPIVersion", func() {
			version, err := c.Version(ctx, nil)
			Expect(err).To(BeNil())
			Expect(version.Version).To(Equal("0.1.0"))
		})

		It("returns the 'RuntimeAPIVersion' = CRI Version", func() {
			version, err := c.Version(ctx, nil)
			Expect(err).To(BeNil())
			Expect(version.RuntimeApiVersion).To(Equal("v1alpha2"))
		})

		It("returns a RuntimeVersion = refunction version", func() {
			version, err := c.Version(ctx, nil)
			Expect(err).To(BeNil())
			Expect(version.RuntimeVersion).NotTo(Equal(""))
		})
	})

	Describe("RunPodSandbox", func() {
		It("returns a non-nil response", func() {
			resp, err := c.RunPodSandbox(ctx, nil)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
		})
	})

	Describe("StopPodSandbox", func() {
		It("returns a non-nil response", func() {
			resp, err := c.StopPodSandbox(ctx, nil)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
		})
	})

	Describe("RemovePodSandbox", func() {
		It("returns a non-nil response", func() {
			resp, err := c.RemovePodSandbox(ctx, nil)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
		})
	})

	Describe("ListPodSandbox", func() {
		It("returns a non-nil response", func() {
			resp, err := c.ListPodSandbox(ctx, nil)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
		})
	})

	Describe("CreateContainer", func() {
		It("returns a non-nil response", func() {
			resp, err := c.CreateContainer(ctx, nil)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
		})
	})

	Describe("StartContainer", func() {
		It("returns a non-nil response", func() {
			resp, err := c.StartContainer(ctx, nil)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
		})
	})

	Describe("StopContainer", func() {
		It("returns a non-nil response", func() {
			resp, err := c.StopContainer(ctx, nil)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
		})
	})

	Describe("RemoveContainer", func() {
		It("returns a non-nil response", func() {
			resp, err := c.RemoveContainer(ctx, nil)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
		})
	})

	Describe("ListContainers", func() {
		It("returns a non-nil response", func() {
			resp, err := c.ListContainers(ctx, nil)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
		})
	})

	Describe("ContainerStatus", func() {
		It("returns a non-nil response", func() {
			resp, err := c.ContainerStatus(ctx, nil)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
		})
	})

	Describe("UpdateContainerResources", func() {
		It("returns a non-nil response", func() {
			resp, err := c.UpdateContainerResources(ctx, nil)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
		})
	})

	Describe("ReopenContainerLog", func() {
		It("returns a non-nil response", func() {
			resp, err := c.ReopenContainerLog(ctx, nil)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
		})
	})

	Describe("ExecSync", func() {
		It("returns a non-nil response", func() {
			resp, err := c.ExecSync(ctx, nil)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
		})
	})

	Describe("Exec", func() {
		It("returns a non-nil response", func() {
			resp, err := c.Exec(ctx, nil)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
		})
	})

	Describe("Attach", func() {
		It("returns a non-nil response", func() {
			resp, err := c.Attach(ctx, nil)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
		})
	})

	Describe("PortForward", func() {
		It("returns a non-nil response", func() {
			resp, err := c.PortForward(ctx, nil)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
		})
	})

	Describe("ContainerStats", func() {
		It("returns a non-nil response", func() {
			resp, err := c.ContainerStats(ctx, nil)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
		})
	})

	Describe("ListContainerStats", func() {
		It("returns a non-nil response", func() {
			resp, err := c.ListContainerStats(ctx, nil)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
		})
	})

	Describe("UpdateRuntimeConfig", func() {
		It("returns a non-nil response", func() {
			resp, err := c.UpdateRuntimeConfig(ctx, nil)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
		})
	})

	Describe("Status", func() {
		It("returns a non-nil response", func() {
			resp, err := c.Status(ctx, nil)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
		})
	})
})

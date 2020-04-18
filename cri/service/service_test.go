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
	var containerdCRI *servicefakes.FakeContainerdCRIService
	ctx := context.Background()

	BeforeEach(func() {
		containerdCRI = new(servicefakes.FakeContainerdCRIService)
		c = NewFakeCRIService(containerdCRI)
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
		It("calls containerd RunSandbox", func() {
			_, err := c.RunPodSandbox(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(containerdCRI.RunPodSandboxCallCount()).To(Equal(1))
		})
	})

	Describe("StopPodSandbox", func() {
		It("calls containerd StopSandBox", func() {
			_, err := c.StopPodSandbox(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(containerdCRI.StopPodSandboxCallCount()).To(Equal(1))
		})
	})

	Describe("RemovePodSandbox", func() {
		It("calls containerd", func() {
			_, err := c.RemovePodSandbox(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(containerdCRI.RemovePodSandboxCallCount()).To(Equal(1))
		})
	})

	Describe("ListPodSandbox", func() {
		It("calls containerd", func() {
			_, err := c.ListPodSandbox(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(containerdCRI.ListPodSandboxCallCount()).To(Equal(1))
		})
	})

	Describe("CreateContainer", func() {
		It("calls containerd", func() {
			_, err := c.CreateContainer(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(containerdCRI.CreateContainerCallCount()).To(Equal(1))
		})
	})

	Describe("StartContainer", func() {
		It("calls containerd", func() {
			_, err := c.StartContainer(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(containerdCRI.StartContainerCallCount()).To(Equal(1))
		})
	})

	Describe("StopContainer", func() {
		It("calls containerd", func() {
			_, err := c.StopContainer(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(containerdCRI.StopContainerCallCount()).To(Equal(1))
		})
	})

	Describe("RemoveContainer", func() {
		It("calls containerd", func() {
			_, err := c.RemoveContainer(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(containerdCRI.RemoveContainerCallCount()).To(Equal(1))
		})
	})

	Describe("ListContainers", func() {
		It("calls containerd", func() {
			_, err := c.ListContainers(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(containerdCRI.ListContainersCallCount()).To(Equal(1))
		})
	})

	Describe("ContainerStatus", func() {
		It("calls containerd", func() {
			_, err := c.ContainerStatus(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(containerdCRI.ContainerStatusCallCount()).To(Equal(1))
		})
	})

	Describe("UpdateContainerResources", func() {
		It("calls containerd", func() {
			_, err := c.UpdateContainerResources(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(containerdCRI.UpdateContainerResourcesCallCount()).To(Equal(1))
		})
	})

	Describe("ReopenContainerLog", func() {
		It("calls containerd", func() {
			_, err := c.ReopenContainerLog(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(containerdCRI.ReopenContainerLogCallCount()).To(Equal(1))
		})
	})

	Describe("ExecSync", func() {
		It("calls containerd", func() {
			_, err := c.ExecSync(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(containerdCRI.ExecSyncCallCount()).To(Equal(1))
		})
	})

	Describe("Exec", func() {
		It("calls containerd", func() {
			_, err := c.Exec(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(containerdCRI.ExecCallCount()).To(Equal(1))
		})
	})

	Describe("Attach", func() {
		It("calls containerd", func() {
			_, err := c.Attach(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(containerdCRI.AttachCallCount()).To(Equal(1))
		})
	})

	Describe("PortForward", func() {
		It("calls containerd", func() {
			_, err := c.PortForward(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(containerdCRI.PortForwardCallCount()).To(Equal(1))
		})
	})

	Describe("ContainerStats", func() {
		It("calls containerd", func() {
			_, err := c.ContainerStats(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(containerdCRI.ContainerStatsCallCount()).To(Equal(1))
		})
	})

	Describe("ListContainerStats", func() {
		It("calls containerd", func() {
			_, err := c.ListContainerStats(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(containerdCRI.ListContainerStatsCallCount()).To(Equal(1))
		})
	})

	Describe("UpdateRuntimeConfig", func() {
		It("calls containerd", func() {
			_, err := c.UpdateRuntimeConfig(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(containerdCRI.UpdateRuntimeConfigCallCount()).To(Equal(1))
		})
	})

	Describe("Status", func() {
		It("calls containerd", func() {
			_, err := c.Status(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(containerdCRI.StatusCallCount()).To(Equal(1))
		})
	})

	Describe("ListImages", func() {
		It("calls containerd", func() {
			_, err := c.ListImages(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(containerdCRI.ListImagesCallCount()).To(Equal(1))
		})
	})

	Describe("ImageStatus", func() {
		It("calls containerd", func() {
			_, err := c.ImageStatus(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(containerdCRI.ImageStatusCallCount()).To(Equal(1))
		})
	})

	Describe("PullImage", func() {
		It("calls containerd", func() {
			_, err := c.PullImage(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(containerdCRI.PullImageCallCount()).To(Equal(1))
		})
	})

	Describe("RemoveImage", func() {
		It("calls containerd", func() {
			_, err := c.RemoveImage(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(containerdCRI.RemoveImageCallCount()).To(Equal(1))
		})
	})

	Describe("ImageFsInfo", func() {
		It("calls containerd", func() {
			_, err := c.ImageFsInfo(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(containerdCRI.ImageFsInfoCallCount()).To(Equal(1))
		})
	})
})

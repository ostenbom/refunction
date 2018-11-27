package worker_test

import (
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/ostenbom/refunction/worker"
)

var _ = Describe("Worker Manager", func() {
	var manager *Manager

	BeforeEach(func() {
		var err error
		id := strconv.Itoa(GinkgoParallelNode())
		manager, err = NewManager(id)
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		err := manager.End()
		Expect(err).To(BeNil())
	})

	It("has an id", func() {
		Expect(manager.Id).NotTo(BeNil())
		Expect(manager.Id).NotTo(Equal(""))
	})

	Describe("CreateContainer", func() {
		BeforeEach(func() {
			manager.StartChild()
		})

		It("pulls an image", func() {
			images, err := manager.ListImages()
			Expect(err).To(BeNil())
			Expect(len(images)).To(Equal(1))
		})

		It("creates a child with a pid", func() {
			pid, err := manager.ChildPid()
			Expect(err).To(BeNil())
			Expect(pid >= 0)
		})

	})

})

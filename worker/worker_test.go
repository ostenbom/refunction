package worker_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/ostenbom/refunction/worker"
)

var _ = Describe("Worker Manager", func() {
	var manager *Manager

	BeforeEach(func() {
		var err error
		manager, err = NewManager()
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		err := manager.End()
		Expect(err).To(BeNil())
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
	})

})

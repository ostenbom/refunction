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
		manager, err = NewManager(id, client)
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
			Expect(manager.StartChild()).To(Succeed())
		})

		It("pulls the ptrace image", func() {
			_, err := manager.GetImage("docker.io/ostenbom/ptrace-sleep:latest")
			Expect(err).To(BeNil())
		})

		It("creates a child with a pid", func() {
			pid, err := manager.ChildPid()
			Expect(err).To(BeNil())
			Expect(pid >= 0)
		})

	})

})

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
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err := manager.End()
		Expect(err).NotTo(HaveOccurred())
	})

	It("has an id", func() {
		Expect(manager.ID).NotTo(BeNil())
		Expect(manager.ID).NotTo(Equal(""))
	})

	Describe("StartChild", func() {
		BeforeEach(func() {
			Expect(manager.StartChild()).To(Succeed())
		})

		It("pulls the ptrace image", func() {
			_, err := manager.GetImage("docker.io/ostenbom/ptrace-sleep:latest")
			Expect(err).NotTo(HaveOccurred())
		})

		It("creates a child with a pid", func() {
			pid, err := manager.ChildPid()
			Expect(err).NotTo(HaveOccurred())
			Expect(pid >= 0)
		})
	})

	Describe("Ptracing", func() {
		BeforeEach(func() {
			Expect(manager.StartChild()).To(Succeed())
		})

		It("Can attach, continue and detach", func() {
			Expect(manager.AttachChild()).To(Succeed())
			Expect(manager.ContinueChild()).To(Succeed())
			Expect(manager.DetachChild()).To(Succeed())
		})
	})

})

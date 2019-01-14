package worker_test

import (
	"os"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/ostenbom/refunction/worker"
)

var _ = Describe("Worker Manager using sigusr-sleep image", func() {
	var manager *Manager
	image := "docker.io/ostenbom/sigusr-sleep:latest"

	BeforeEach(func() {
		var err error
		id := strconv.Itoa(GinkgoParallelNode())
		manager, err = NewManager(id, client, "sigusr-sleep", image)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err := manager.End()
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("StartChild - sigusr-sleep", func() {
		BeforeEach(func() {
			Expect(manager.StartChild()).To(Succeed())
		})

		It("creates a child with a pid", func() {
			pid, err := manager.ChildPid()
			Expect(err).NotTo(HaveOccurred())
			Expect(pid >= 0)
		})

		It("does not create the count file on start", func() {
			countLocation := config.State + "/io.containerd.runtime.v1.linux/refunction-worker1/sigusr-sleep-1/rootfs/count.txt"

			if _, err := os.Stat(countLocation); !os.IsNotExist(err) {
				Fail("count file exists without SIGUSR1")
			}
		})

		It("creates the count file after SIGUSR1", func() {
			// Send custom "ready" signal to container
			err := manager.SendEnableSignal()
			Expect(err).NotTo(HaveOccurred())

			countLocation := config.State + "/io.containerd.runtime.v1.linux/refunction-worker1/sigusr-sleep-1/rootfs/count.txt"

			Eventually(func() bool {
				_, err := os.Stat(countLocation)
				return os.IsNotExist(err)
			}).Should(BeFalse())
		})
	})

	Describe("Ptracing", func() {
		BeforeEach(func() {
			Expect(manager.StartChild()).To(Succeed())
		})

		It("can attach and detach", func() {
			Expect(manager.AttachChild()).To(Succeed())
			Expect(manager.DetachChild()).To(Succeed())
		})

		It("is in a stopped state after attaching", func() {
			Expect(manager.AttachChild()).To(Succeed())

			pid, err := manager.ChildPid()
			Expect(err).NotTo(HaveOccurred())

			processState := getPidState(pid)

			// t = stopped by debugger. T = stopped by signal
			Expect(strings.Contains(processState, "t")).To(BeTrue())

			Expect(manager.DetachChild()).To(Succeed())
		})

		It("creates a count file if allowed to continue, given SIGUSR1", func() {
			Expect(manager.AttachChild()).To(Succeed())
			Expect(manager.ContinueChild()).To(Succeed())

			err := manager.SendEnableSignal()
			Expect(err).NotTo(HaveOccurred())

			countLocation := config.State + "/io.containerd.runtime.v1.linux/refunction-worker1/sigusr-sleep-1/rootfs/count.txt"

			Eventually(func() bool {
				_, err := os.Stat(countLocation)
				return os.IsNotExist(err)
			}).Should(BeFalse())

			Expect(manager.StopDetachChild()).To(Succeed())
		})
	})

})

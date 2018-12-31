package worker_test

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

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

	Describe("StartChild - ptrace-sleep", func() {
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

		It("does not create the count file", func() {
			countLocation := config.State + "/io.containerd.runtime.v1.linux/refunction-worker1/ptrace-sleep-1/rootfs/home/count.txt"

			if _, err := os.Stat(countLocation); !os.IsNotExist(err) {
				Fail("count file exists without SIGUSR1")
			}
		})

		It("creates the count file after SIGUSR1", func() {
			pid, err := manager.ChildPid()
			Expect(err).NotTo(HaveOccurred())

			// Send custom "ready" signal to container
			err = syscall.Kill(int(pid), syscall.SIGUSR1)
			Expect(err).NotTo(HaveOccurred())

			countLocation := config.State + "/io.containerd.runtime.v1.linux/refunction-worker1/ptrace-sleep-1/rootfs/count.txt"

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

		It("creates the process in a stopped state after attaching", func() {
			Expect(manager.AttachChild()).To(Succeed())

			pid, err := manager.ChildPid()
			Expect(err).NotTo(HaveOccurred())

			processState := getPidState(pid)

			// t = stopped by debugger. T = stopped by signal
			Expect(strings.Contains(processState, "t")).To(BeTrue())

			Expect(manager.DetachChild()).To(Succeed())
		})
	})

})

func getPidState(pid uint32) string {
	psArgs := []string{"-p", strconv.Itoa(int(pid)), "-o", "stat="}
	processState, err := exec.Command("ps", psArgs...).Output()
	Expect(err).NotTo(HaveOccurred())

	return string(processState)
}

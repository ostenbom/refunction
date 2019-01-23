package worker_test

import (
	"os"
	"os/exec"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/ostenbom/refunction/worker"
)

var _ = Describe("Worker Manager using go-ptrace-sleep image", func() {
	var manager *Worker
	image := "go-ptrace-sleep"

	BeforeEach(func() {
		var err error
		id := strconv.Itoa(GinkgoParallelNode())
		manager, err = NewWorker(id, client, "go-ptrace-sleep", image)
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

	Describe("StartChild - go-ptrace-sleep", func() {
		BeforeEach(func() {
			Expect(manager.StartChild()).To(Succeed())
		})

		It("creates a child with a pid", func() {
			pid, err := manager.ChildPid()
			Expect(err).NotTo(HaveOccurred())
			Expect(pid >= 0)
		})

		It("does not create the count file on start", func() {
			countLocation := getRootfs(manager, "go-ptrace-sleep") + "count.txt"

			if _, err := os.Stat(countLocation); !os.IsNotExist(err) {
				Fail("count file exists without SIGUSR1")
			}
		})

		It("creates the count file after SIGUSR1", func() {
			// Send custom "ready" signal to container
			err := manager.SendEnableSignal()
			Expect(err).NotTo(HaveOccurred())

			countLocation := getRootfs(manager, "go-ptrace-sleep") + "count.txt"

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
			Expect(manager.Attach()).To(Succeed())
			Expect(manager.Detach()).To(Succeed())
		})

		It("is in a stopped state after attaching", func() {
			Expect(manager.Attach()).To(Succeed())

			pid, err := manager.ChildPid()
			Expect(err).NotTo(HaveOccurred())

			processState := getPidState(pid)

			// t = stopped by debugger. T = stopped by signal
			Expect(strings.Contains(processState, "t")).To(BeTrue())

			Expect(manager.Detach()).To(Succeed())
		})
	})

})

func getPidState(pid uint32) string {
	psArgs := []string{"-p", strconv.Itoa(int(pid)), "-o", "stat="}
	processState, err := exec.Command("ps", psArgs...).Output()
	Expect(err).NotTo(HaveOccurred())

	return string(processState)
}

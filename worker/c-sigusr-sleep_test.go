package worker_test

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/ostenbom/refunction/worker"
)

var _ = Describe("Worker Manager using c-sigusr-sleep image", func() {
	var worker *Worker
	image := "c-sigusr-sleep"

	BeforeEach(func() {
		var err error
		id := strconv.Itoa(GinkgoParallelNode())
		worker, err = NewWorker(id, client, image)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err := worker.End()
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("StartChild - c-sigusr-sleep", func() {
		BeforeEach(func() {
			Expect(worker.Start()).To(Succeed())
		})

		It("creates a child with a pid", func() {
			pid, err := worker.Pid()
			Expect(err).NotTo(HaveOccurred())
			Expect(pid >= 0)
		})

		It("does not create the count file on start", func() {
			countLocation := getRootfs(worker, "c-sigusr-sleep") + "count.txt"

			if _, err := os.Stat(countLocation); !os.IsNotExist(err) {
				Fail("count file exists without SIGUSR1")
			}
		})

		It("creates the count file after SIGUSR1", func() {
			// Send custom "ready" signal to container
			err := worker.SendEnableSignal()
			Expect(err).NotTo(HaveOccurred())

			countLocation := getRootfs(worker, "c-sigusr-sleep") + "count.txt"

			Eventually(func() bool {
				_, err := os.Stat(countLocation)
				return os.IsNotExist(err)
			}).Should(BeFalse())
		})
	})

	Describe("Ptracing", func() {
		BeforeEach(func() {
			Expect(worker.Start()).To(Succeed())
		})

		It("can attach and detach", func() {
			Expect(worker.Attach()).To(Succeed())
			Expect(worker.Detach()).To(Succeed())
		})

		It("is in a stopped state after attaching", func() {
			Expect(worker.Attach()).To(Succeed())

			pid, err := worker.Pid()
			Expect(err).NotTo(HaveOccurred())

			processState := getPidState(pid)

			// t = stopped by debugger. T = stopped by signal
			Expect(strings.Contains(processState, "t")).To(BeTrue())

			Expect(worker.Detach()).To(Succeed())
		})

		It("creates a count file if allowed to continue, given SIGUSR1", func() {
			Expect(worker.Attach()).To(Succeed())
			Expect(worker.Continue()).To(Succeed())

			err := worker.SendEnableSignal()
			Expect(err).NotTo(HaveOccurred())

			countLocation := getRootfs(worker, "c-sigusr-sleep") + "count.txt"

			Eventually(func() bool {
				_, err = os.Stat(countLocation)
				return os.IsNotExist(err)
			}).Should(BeFalse())

			err = worker.Detach()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

func getRootfs(manager *Worker, imageName string) string {
	return fmt.Sprintf("%s/io.containerd.runtime.v1.linux/refunction-worker%s/%s-%s/rootfs/", config.State, manager.ID, imageName, manager.ID)
}

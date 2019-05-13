package worker_test

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/ostenbom/refunction/worker"
)

var _ = Describe("Worker Manager using python runtime", func() {
	var worker *Worker
	runtime := "python"
	image := "sigusr-sleep.py"

	BeforeEach(func() {
		var err error
		id := strconv.Itoa(GinkgoParallelNode())
		worker, err = NewWorker(id, client, runtime, image)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err := worker.End()
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("StartChild - sigusr-sleep.py", func() {
		BeforeEach(func() {
			Expect(worker.Start()).To(Succeed())
		})

		It("creates a child with a pid", func() {
			pid := worker.Pid()
			Expect(pid >= 0)
		})

		It("does not create the count file on start", func() {
			countLocation := getRootfs(worker) + "tmp/count.txt"

			if _, err := os.Stat(countLocation); !os.IsNotExist(err) {
				Fail("count file exists without SIGUSR1")
			}
		})

		It("creates the count file after SIGUSR1", func() {
			Expect(worker.Attach()).To(Succeed())
			defer worker.Detach()
			worker.AwaitSignal(syscall.SIGUSR2)

			// Send custom "ready" signal to container
			err := worker.SendSignalCont(syscall.SIGUSR1)
			Expect(err).NotTo(HaveOccurred())

			countLocation := getRootfs(worker) + "tmp/count.txt"

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

		It("is in a ptrace-stopped state after attaching and sending signal", func() {
			Expect(worker.Attach()).To(Succeed())
			defer worker.Detach()

			pid := worker.Pid()

			Expect(worker.SendSignal(syscall.SIGSTOP)).To(Succeed())
			processState := getPidState(pid)

			// t = stopped by debugger. T = stopped by signal
			Expect(strings.Contains(processState, "t")).To(BeTrue())

		})

		It("creates a count file if allowed to continue, given SIGUSR1", func() {
			Expect(worker.Attach()).To(Succeed())
			defer worker.Detach()

			worker.AwaitSignal(syscall.SIGUSR2)

			err := worker.SendSignalCont(syscall.SIGUSR1)
			Expect(err).NotTo(HaveOccurred())

			countLocation := getRootfs(worker) + "tmp/count.txt"

			Eventually(func() bool {
				_, err = os.Stat(countLocation)
				return os.IsNotExist(err)
			}).Should(BeFalse())
		})
	})
})

var _ = Describe("Worker Manager using c-sigusr-sleep image", func() {
	var worker *Worker
	runtime := "alpine"
	image := "c-sigusr-sleep"

	BeforeEach(func() {
		var err error
		id := strconv.Itoa(GinkgoParallelNode())
		worker, err = NewWorker(id, client, runtime, image)
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
			pid := worker.Pid()
			Expect(pid >= 0)
		})

		It("does not create the count file on start", func() {
			countLocation := getRootfs(worker) + "tmp/count.txt"

			if _, err := os.Stat(countLocation); !os.IsNotExist(err) {
				Fail("count file exists without SIGUSR1")
			}
		})

		It("creates the count file after SIGUSR1", func() {
			// Send custom "ready" signal to container
			err := worker.SendSignalCont(syscall.SIGUSR1)
			Expect(err).NotTo(HaveOccurred())

			countLocation := getRootfs(worker) + "tmp/count.txt"

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

		It("is in a ptrace-stop state after attach and signal", func() {
			Expect(worker.Attach()).To(Succeed())
			defer worker.Detach()

			err := worker.SendSignalCont(syscall.SIGUSR1)
			Expect(err).NotTo(HaveOccurred())

			pid := worker.Pid()
			Expect(err).NotTo(HaveOccurred())

			worker.SendSignal(syscall.SIGSTOP)
			// worker.PauseAtSignal(syscall.SIGSTOP)
			processState := getPidState(pid)

			// t = stopped by debugger. T = stopped by signal
			Expect(strings.Contains(processState, "t")).To(BeTrue())

			// time.Sleep(time.Second * 30)
		})

		It("creates a count file if allowed to continue, given SIGUSR1", func() {
			Expect(worker.Attach()).To(Succeed())
			defer worker.Detach()

			err := worker.SendSignalCont(syscall.SIGUSR1)
			Expect(err).NotTo(HaveOccurred())

			countLocation := getRootfs(worker) + "tmp/count.txt"

			Eventually(func() bool {
				_, err = os.Stat(countLocation)
				return os.IsNotExist(err)
			}).Should(BeFalse())
		})
	})
})

var _ = Describe("Worker Manager using go-ptrace-sleep image", func() {
	var manager *Worker
	runtime := "alpine"
	image := "go-ptrace-sleep"

	BeforeEach(func() {
		var err error
		id := strconv.Itoa(GinkgoParallelNode())
		manager, err = NewWorker(id, client, runtime, image)
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
			Expect(manager.Start()).To(Succeed())
		})

		It("creates a child with a pid", func() {
			pid := manager.Pid()
			Expect(pid >= 0)
		})

		It("does not create the count file on start", func() {
			countLocation := getRootfs(manager) + "tmp/count.txt"

			if _, err := os.Stat(countLocation); !os.IsNotExist(err) {
				Fail("count file exists without SIGUSR1")
			}
		})

		It("creates the count file after go message", func() {
			manager.AwaitMessage("started")
			err := manager.SendMessage("go", "")
			Expect(err).NotTo(HaveOccurred())

			countLocation := getRootfs(manager) + "tmp/count.txt"

			Eventually(func() bool {
				_, err := os.Stat(countLocation)
				return os.IsNotExist(err)
			}).Should(BeFalse())
		})
	})

	Describe("Ptracing", func() {
		BeforeEach(func() {
			Expect(manager.Start()).To(Succeed())
		})

		It("can attach and detach", func() {
			Expect(manager.Attach()).To(Succeed())
			Expect(manager.Detach()).To(Succeed())
		})

		It("is in a stopped state after attaching and sending a signal", func() {
			Expect(manager.Attach()).To(Succeed())
			defer manager.Detach()

			pid := manager.Pid()

			Expect(manager.SendSignal(syscall.SIGSTOP)).To(Succeed())
			processState := getPidState(pid)

			// t = stopped by debugger. T = stopped by signal
			Expect(strings.Contains(processState, "t")).To(BeTrue())
		})
	})

})

func getPidState(pid int) string {
	psArgs := []string{"-p", strconv.Itoa(int(pid)), "-o", "stat="}
	processState, err := exec.Command("ps", psArgs...).Output()
	Expect(err).NotTo(HaveOccurred())

	return string(processState)
}

func getRootfs(manager *Worker) string {
	return fmt.Sprintf("%s/io.containerd.runtime.v1.linux/refunction-worker%s/%s/rootfs/", config.State, manager.ID, manager.ContainerID)
}

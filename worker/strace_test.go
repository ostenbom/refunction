package worker_test

import (
	"io"
	"os"
	"strconv"
	"syscall"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	. "github.com/ostenbom/refunction/worker"
)

var _ = Describe("Worker Manager syscall tracing c-sigusr-sleep image", func() {
	var worker *Worker
	runtime := "alpine"
	image := "c-sigusr-sleep"
	var straceBuffer *gbytes.Buffer

	BeforeEach(func() {
		var err error
		id := strconv.Itoa(GinkgoParallelNode())
		worker, err = NewWorker(id, client, runtime, image)
		Expect(err).NotTo(HaveOccurred())

		straceBuffer = gbytes.NewBuffer()
		multiBuffer := io.MultiWriter(straceBuffer, GinkgoWriter)
		worker.WithSyscallTrace(multiBuffer)
	})

	AfterEach(func() {
		err := worker.End()
		Expect(err).NotTo(HaveOccurred())
	})

	BeforeEach(func() {
		Expect(worker.Start()).To(Succeed())
	})

	It("prints syscalls", func() {
		Expect(worker.Attach()).To(Succeed())
		defer worker.Detach()
		worker.SendSignalCont(syscall.SIGUSR1)

		countLocation := getRootfs(worker) + "tmp/count.txt"
		Eventually(func() bool {
			_, err := os.Stat(countLocation)
			return os.IsNotExist(err)
		}).Should(BeFalse())

		Eventually(straceBuffer).Should(gbytes.Say("syscall"))
	})

})

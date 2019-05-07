package worker_test

import (
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	. "github.com/ostenbom/refunction/worker"
)

var _ = Describe("Worker", func() {
	var id string

	BeforeEach(func() {
		id = strconv.Itoa(GinkgoParallelNode())
	})

	Describe("io streams", func() {
		var worker *Worker
		var targetLayer string
		var stdout *gbytes.Buffer
		var stderr *gbytes.Buffer
		runtime := "alpine"

		Context("when writing to stdout", func() {
			BeforeEach(func() {
				var err error
				targetLayer = "echo-hello"
				worker, err = NewWorker(id, client, runtime, targetLayer)
				Expect(err).NotTo(HaveOccurred())
				stdout = gbytes.NewBuffer()
				stderr = gbytes.NewBuffer()
				worker.WithStdPipes(stderr, stdout)

				Expect(worker.Start()).To(Succeed())
			})

			It("can read stdout", func() {
				Eventually(stdout).Should(gbytes.Say("hello!"))
			})
		})

		Context("when writing to stderr", func() {
			BeforeEach(func() {
				var err error
				targetLayer = "echo-error"
				worker, err = NewWorker(id, client, runtime, targetLayer)
				Expect(err).NotTo(HaveOccurred())
				stdout = gbytes.NewBuffer()
				stderr = gbytes.NewBuffer()
				worker.WithStdPipes(stderr, stdout)

				Expect(worker.Start()).To(Succeed())
			})

			It("can read stderr", func() {
				Eventually(stderr).Should(gbytes.Say("error!"))
			})
		})

		AfterEach(func() {
			err := worker.End()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

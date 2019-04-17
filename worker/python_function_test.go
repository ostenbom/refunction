package worker_test

import (
	"io"
	"strconv"
	"syscall"

	"github.com/containerd/containerd/cio"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	. "github.com/ostenbom/refunction/worker"
)

var _ = Describe("Python Serverless Function Management", func() {
	var id string
	runtime := "python"
	var targetLayer string
	var worker *Worker
	var stdout *gbytes.Buffer
	var stderr *gbytes.Buffer

	BeforeEach(func() {
		id = strconv.Itoa(GinkgoParallelNode())
	})

	JustBeforeEach(func() {
		var err error
		worker, err = NewWorker(id, client, runtime, targetLayer)
		Expect(err).NotTo(HaveOccurred())
		stdout = gbytes.NewBuffer()
		stderr = gbytes.NewBuffer()
		worker.WithCreator(cio.NewCreator(cio.WithStreams(nil, io.MultiWriter(stdout, GinkgoWriter), io.MultiWriter(stderr, GinkgoWriter))))

		Expect(worker.Start()).To(Succeed())
	})

	AfterEach(func() {
		err := worker.End()
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("simple single function run", func() {
		BeforeEach(func() {
			targetLayer = "single-function.py"
		})

		It("can reset after done signal", func() {
			Expect(worker.Activate()).To(Succeed())

			Eventually(stdout).Should(gbytes.Say("handling request"))

			worker.PauseAtSignal(syscall.SIGUSR2)
			Expect(worker.Restore()).To(Succeed())
			worker.Continue()

			Eventually(stdout).Should(gbytes.Say("handling request"))
		})
	})

	Describe("server managed function", func() {

		BeforeEach(func() {
			targetLayer = "serverless-function.py"
		})

		It("can load a function and send a request", func() {
			// Initiate python ready sequence
			Expect(worker.Activate()).To(Succeed())
			Expect(len(worker.GetCheckpoints())).To(Equal(1))
			Eventually(stdout).Should(gbytes.Say("loading function"))

			function := "def handle(req):\n  print(req)\n  return req"
			Expect(worker.SendFunction(function)).To(Succeed())
			Eventually(stdout).Should(gbytes.Say("starting function server"))

			request := "{\"greatkey\": \"nicevalue\"}"
			requestInPython := "{'greatkey': 'nicevalue'}"
			_, err := worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Eventually(stdout).Should(gbytes.Say(requestInPython))
		})

		It("can get a request response", func() {
			Expect(worker.Activate()).To(Succeed())

			function := "def handle(req):\n  print(req)\n  return req"
			Expect(worker.SendFunction(function)).To(Succeed())

			request := "{\"greatkey\": \"nicevalue\"}"
			response, err := worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal(request))
		})

		It("can get several request responses", func() {
			Expect(worker.Activate()).To(Succeed())

			function := "def handle(req):\n  print(req)\n  return req"
			Expect(worker.SendFunction(function)).To(Succeed())

			request := "{\"greatkey\": \"nicevalue\"}"
			response, err := worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(request).To(Equal(response))

			request = "{\"greatkey2\": \"nicevalue\"}"
			response, err = worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(request).To(Equal(response))

			request = "{\"greatkey3\": \"nicevalue\"}"
			response, err = worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(request).To(Equal(response))
		})

	})
})

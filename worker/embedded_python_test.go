package worker_test

import (
	"io"
	"strconv"
	"syscall"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	. "github.com/ostenbom/refunction/worker"
)

var _ = Describe("Embedded Python Serverless Function Management", func() {
	var id string
	// runtime := "python3-dbg"
	runtime := "alpinepython"
	var targetLayer string
	var worker *Worker
	var stdout *gbytes.Buffer
	var straceBuffer *gbytes.Buffer

	BeforeEach(func() {
		id = strconv.Itoa(GinkgoParallelNode())
	})

	JustBeforeEach(func() {
		var err error
		worker, err = NewWorker(id, client, runtime, targetLayer)
		Expect(err).NotTo(HaveOccurred())
		stdout = gbytes.NewBuffer()
		worker.WithStdPipeCommunication(GinkgoWriter, stdout, GinkgoWriter)

		straceBuffer = gbytes.NewBuffer()
		multiBuffer := io.MultiWriter(straceBuffer, GinkgoWriter)
		worker.WithSyscallTrace(multiBuffer)

		Expect(worker.Start()).To(Succeed())
	})

	AfterEach(func() {
		err := worker.End()
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("server managed functions", func() {

		BeforeEach(func() {
			targetLayer = "embedded-python"
		})

		It("can load a function", func() {
			// Initiate python ready sequence
			Expect(worker.Activate()).To(Succeed())
			Expect(len(worker.GetCheckpoints())).To(Equal(1))
			Eventually(stdout).Should(gbytes.Say("python started"))
			Eventually(stdout).Should(gbytes.Say("post checkpoint"))

			function := "def handle(req):\n  print(req)\n  return req"
			Expect(worker.SendFunction(function)).To(Succeed())
			Eventually(stdout).Should(gbytes.Say("handle function successfully loaded"))
		})

		It("can get a request response", func() {
			Expect(worker.Activate()).To(Succeed())

			function := "def handle(req):\n  print(req)\n  return req"
			Expect(worker.SendFunction(function)).To(Succeed())

			request := "\"jsonstring\""
			response, err := worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal(request))
		})

		It("can get an object request response", func() {
			Expect(worker.Activate()).To(Succeed())

			function := "def handle(req):\n  print(req)\n  return req"
			Expect(worker.SendFunction(function)).To(Succeed())

			request := "{\"greatkey\":\"nicevalue\"}"
			response, err := worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal(request))
		})

		It("can get several request responses", func() {
			Expect(worker.Activate()).To(Succeed())

			function := "def handle(req):\n  print(req)\n  return req"
			Expect(worker.SendFunction(function)).To(Succeed())
			Eventually(stdout).Should(gbytes.Say("handle function successfully loaded"))

			request := "\"jsonstring\""
			response, err := worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal(request))

			request = "\"anotherstring\""
			response, err = worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal(request))

			request = "\"whateverstring\""
			response, err = worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal(request))
		})

		It("can restore and change function", func() {
			Expect(worker.Activate()).To(Succeed())

			function := "def handle(req):\n  print(req)\n  return req"
			Expect(worker.SendFunction(function)).To(Succeed())

			request := "\"jsonstring\""
			response, err := worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal(request))

			Expect(worker.SendSignal(syscall.SIGUSR2)).To(Succeed())
			worker.AwaitSignal(syscall.SIGUSR2)

			Expect(worker.Restore()).To(Succeed())

			function = "def handle(req):\n  print(req)\n  return 'unrelated'"
			Expect(worker.SendFunction(function)).To(Succeed())

			request = "\"anotherstring\""
			response, err = worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal("\"unrelated\""))
		})

		It("can load a function with an import from the std library", func() {
			Expect(worker.Activate()).To(Succeed())

			function := "import math\ndef handle(req):\n  print(req)\n  return math.ceil(req)"
			Expect(worker.SendFunction(function)).To(Succeed())

			request := "3.5"
			response, err := worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal("4"))
		})

		It("can load different stdlibrary functions", func() {
			Expect(worker.Activate()).To(Succeed())

			function := "import math\ndef handle(req):\n  print(req)\n  return math.ceil(req)"
			Expect(worker.SendFunction(function)).To(Succeed())

			request := "3.5"
			response, err := worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal("4"))

			Expect(worker.SendSignal(syscall.SIGUSR2)).To(Succeed())
			worker.AwaitSignal(syscall.SIGUSR2)

			Expect(worker.Restore()).To(Succeed())

			function = "import string\ndef handle(req):\n  print(req)\n  return string.ascii_lowercase"
			Expect(worker.SendFunction(function)).To(Succeed())

			request = "\"dummyanything\""
			response, err = worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal("\"abcdefghijklmnopqrstuvwxyz\""))
		})
	})
})

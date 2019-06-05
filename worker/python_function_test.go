package worker_test

import (
	"io"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	. "github.com/ostenbom/refunction/worker"
)

var _ = Describe("Python Serverless Function Management", func() {
	var id string
	// runtime := "python3-dbg"
	runtime := "python"
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
		worker.WithStdPipes(GinkgoWriter, stdout, GinkgoWriter)

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
			targetLayer = "serverless-function.py"
		})

		It("can load a function", func() {
			// Initiate python ready sequence
			Expect(worker.Activate()).To(Succeed())
			Expect(len(worker.GetCheckpoints())).To(Equal(1))
			Eventually(stdout).Should(gbytes.Say("started"))

			function := "def main(req):\n  print(req)\n  return req"
			Expect(worker.SendFunction(function)).To(Succeed())
			Eventually(stdout).Should(gbytes.Say("{\"type\": \"function_loaded\", \"data\": true}"))
		})

		It("can get a request response", func() {
			Expect(worker.Activate()).To(Succeed())

			function := "def main(req):\n  print(req)\n  return req"
			Expect(worker.SendFunction(function)).To(Succeed())

			request := "jsonstring"
			response, err := worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal(request))
		})

		It("can get an object request response", func() {
			Expect(worker.Activate()).To(Succeed())

			function := "def main(req):\n  print(req)\n  return req"
			Expect(worker.SendFunction(function)).To(Succeed())

			request := map[string]interface{}{
				"greatkey": "nicevalue",
			}
			response, err := worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal(request))
		})

		It("can get several request responses", func() {
			Expect(worker.Activate()).To(Succeed())

			function := "def main(req):\n  print(req)\n  return req"
			Expect(worker.SendFunction(function)).To(Succeed())
			Eventually(stdout).Should(gbytes.Say("{\"type\": \"function_loaded\", \"data\": true}"))

			request := "jsonstring"
			response, err := worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal(request))

			request = "anotherstring"
			response, err = worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal(request))

			request = "whateverstring"
			response, err = worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal(request))
		})

		It("can restore and change function", func() {
			Expect(worker.Activate()).To(Succeed())

			function := "def main(req):\n  print(req)\n  return req"
			Expect(worker.SendFunction(function)).To(Succeed())

			request := "jsonstring"
			response, err := worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal(request))

			Expect(worker.Restore()).To(Succeed())

			function = "def main(req):\n  print(req)\n  return 'unrelated'"
			Expect(worker.SendFunction(function)).To(Succeed())

			request = "anotherstring"
			response, err = worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal("unrelated"))
		})

		It("can load a function with an import from the std library", func() {
			Expect(worker.Activate()).To(Succeed())

			function := "import math\ndef main(req):\n  print(req)\n  return math.ceil(req)"
			Expect(worker.SendFunction(function)).To(Succeed())

			request := 3.5
			response, err := worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			switch v := response.(type) {
			case float64:
				Expect(v).To(Equal(float64(4)))
			default:
				Fail("function returned unknown type")
			}
		})

		It("can load different stdlibrary functions", func() {
			Expect(worker.Activate()).To(Succeed())

			function := "import math\ndef main(req):\n  print(req)\n  return math.ceil(req)"
			Expect(worker.SendFunction(function)).To(Succeed())

			request := 3.5
			response, err := worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			switch v := response.(type) {
			case float64:
				Expect(v).To(Equal(float64(4)))
			default:
				Fail("function returned unknown type")
			}

			Expect(worker.Restore()).To(Succeed())

			function = "import string\ndef main(req):\n  print(req)\n  return string.ascii_lowercase"
			Expect(worker.SendFunction(function)).To(Succeed())

			newrequest := "dummyanything"
			response, err = worker.SendRequest(newrequest)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal("abcdefghijklmnopqrstuvwxyz"))
		})

		It("is resiliant to improper function loads", func() {
			Expect(worker.Activate()).To(Succeed())

			// JS for example
			function := "function main(params) {\n    return params || {};\n}\n"
			err := worker.SendFunction(function)
			Expect(err).NotTo(BeNil())
			Eventually(stdout).Should(gbytes.Say("{\"type\": \"function_loaded\", \"data\": false}"))

			function = "def main(req):\n  print(req)\n  return req"
			Expect(worker.SendFunction(function)).To(Succeed())
			Eventually(stdout).Should(gbytes.Say("{\"type\": \"function_loaded\", \"data\": true}"))

			request := "jsonstring"
			response, err := worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal(request))
		})

		It("can see if the program break has changed", func() {
			longStringFunc := `
import random
import string

def randomString(stringLength=10):
		letters = string.ascii_lowercase
		return ''.join(random.choice(letters) for i in range(stringLength))

def main(params):
		strings = []
		for i in range(5):
				strings.append(randomString(100000))
		to_return = random.randint(0, 4)
		return strings[to_return]
`

			Expect(worker.Activate()).To(Succeed())
			Expect(worker.SendFunction(longStringFunc)).To(Succeed())
			response, err := worker.SendRequest("")
			Expect(err).NotTo(HaveOccurred())

			Expect(len(response.(string))).To(Equal(100000))

			state, err := worker.GetInitialCheckpoint()
			Expect(err).NotTo(HaveOccurred())

			Expect(state.MemoryChanged()).To(BeTrue())
			Expect(state.ProgramBreakChanged()).To(BeTrue())
		})
	})
})

package worker_test

import (
	"io"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	. "github.com/ostenbom/refunction/worker"
)

var _ = Describe("Java Serverless Function Management", func() {
	var id string
	runtime := "java"
	var targetLayer string
	var worker *Worker
	var stdout *gbytes.Buffer
	var straceBuffer *gbytes.Buffer

	var echoFunction = "yv66vgAAADQAIQoABgAPCQAQABEKABIAEwoAFAAVBwAWBwAXAQAGPGluaXQ+AQADKClWAQAEQ29kZQEAD0xpbmVOdW1iZXJUYWJsZQEABG1haW4BADooTGNvbS9nb29nbGUvZ3Nvbi9Kc29uT2JqZWN0OylMY29tL2dvb2dsZS9nc29uL0pzb25PYmplY3Q7AQAKU291cmNlRmlsZQEADUZ1bmN0aW9uLmphdmEMAAcACAcAGAwAGQAaBwAbDAAcAB0HAB4MAB8AIAEACEZ1bmN0aW9uAQAQamF2YS9sYW5nL09iamVjdAEAEGphdmEvbGFuZy9TeXN0ZW0BAANvdXQBABVMamF2YS9pby9QcmludFN0cmVhbTsBABpjb20vZ29vZ2xlL2dzb24vSnNvbk9iamVjdAEACHRvU3RyaW5nAQAUKClMamF2YS9sYW5nL1N0cmluZzsBABNqYXZhL2lvL1ByaW50U3RyZWFtAQAHcHJpbnRsbgEAFShMamF2YS9sYW5nL1N0cmluZzspVgAhAAUABgAAAAAAAgABAAcACAABAAkAAAAdAAEAAQAAAAUqtwABsQAAAAEACgAAAAYAAQAAAAMACQALAAwAAQAJAAAAKAACAAEAAAAMsgACKrYAA7YABCqwAAAAAQAKAAAACgACAAAABQAKAAYAAQANAAAAAgAO"

	var yoloSwagFunction = "yv66vgAAADQAKgoACwAUCQAVABYKAAUAFwoAGAAZBwAaCgAFABQIABsIABwKAAUAHQcAHgcAHwEABjxpbml0PgEAAygpVgEABENvZGUBAA9MaW5lTnVtYmVyVGFibGUBAARtYWluAQA6KExjb20vZ29vZ2xlL2dzb24vSnNvbk9iamVjdDspTGNvbS9nb29nbGUvZ3Nvbi9Kc29uT2JqZWN0OwEAClNvdXJjZUZpbGUBAA1GdW5jdGlvbi5qYXZhDAAMAA0HACAMACEAIgwAIwAkBwAlDAAmACcBABpjb20vZ29vZ2xlL2dzb24vSnNvbk9iamVjdAEABHlvbG8BAARzd2FnDAAoACkBAAhGdW5jdGlvbgEAEGphdmEvbGFuZy9PYmplY3QBABBqYXZhL2xhbmcvU3lzdGVtAQADb3V0AQAVTGphdmEvaW8vUHJpbnRTdHJlYW07AQAIdG9TdHJpbmcBABQoKUxqYXZhL2xhbmcvU3RyaW5nOwEAE2phdmEvaW8vUHJpbnRTdHJlYW0BAAdwcmludGxuAQAVKExqYXZhL2xhbmcvU3RyaW5nOylWAQALYWRkUHJvcGVydHkBACcoTGphdmEvbGFuZy9TdHJpbmc7TGphdmEvbGFuZy9TdHJpbmc7KVYAIQAKAAsAAAAAAAIAAQAMAA0AAQAOAAAAHQABAAEAAAAFKrcAAbEAAAABAA8AAAAGAAEAAAADAAkAEAARAAEADgAAAEAAAwACAAAAHLIAAiq2AAO2AAS7AAVZtwAGTCsSBxIItgAJK7AAAAABAA8AAAASAAQAAAAFAAoABgASAAcAGgAIAAEAEgAAAAIAEw=="

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
			targetLayer = "serverless-java"
		})

		It("can load a function", func() {
			// Initiate python ready sequence
			Expect(worker.Activate()).To(Succeed())
			Expect(len(worker.GetCheckpoints())).To(Equal(1))
			Eventually(stdout).Should(gbytes.Say("started"))

			Expect(worker.SendFunction(echoFunction)).To(Succeed())
			Eventually(stdout).Should(gbytes.Say("{\"type\":\"function_loaded\",\"data\":true}"))
		})

		It("can get an object request response", func() {
			Expect(worker.Activate()).To(Succeed())

			Expect(worker.SendFunction(echoFunction)).To(Succeed())

			request := map[string]interface{}{
				"greatkey": "nicevalue",
			}
			response, err := worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal(request))
		})

		It("can get several request responses", func() {
			Expect(worker.Activate()).To(Succeed())

			Expect(worker.SendFunction(echoFunction)).To(Succeed())
			Eventually(stdout).Should(gbytes.Say("{\"type\":\"function_loaded\",\"data\":true}"))

			request := map[string]interface{}{
				"greatkey": "nicevalue",
			}
			response, err := worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal(request))

			request = map[string]interface{}{
				"otherkey": "radvalue",
			}
			response, err = worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal(request))

			request = map[string]interface{}{
				"somethingelse": "wickedvalue",
			}
			response, err = worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal(request))
		})

		It("can restore and change function", func() {
			Expect(worker.Activate()).To(Succeed())

			Expect(worker.SendFunction(echoFunction)).To(Succeed())

			request := map[string]interface{}{
				"greatkey": "nicevalue",
			}
			response, err := worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal(request))

			Expect(worker.Restore()).To(Succeed())

			Expect(worker.SendFunction(yoloSwagFunction)).To(Succeed())

			expectedResponse := map[string]interface{}{
				"yolo": "swag",
			}
			response, err = worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal(expectedResponse))
		})

		XIt("can load a function with an import from the std library", func() {
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

		XIt("can load different stdlibrary functions", func() {
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

		XIt("is resiliant to improper function loads", func() {
			Expect(worker.Activate()).To(Succeed())

			// JS for example
			function := "function main(params) {\n    return params || {};\n}\n"
			err := worker.SendFunction(function)
			Expect(err).NotTo(BeNil())
			Eventually(stdout).Should(gbytes.Say("{\"data\":false,\"type\":\"function_loaded\"}"))

			function = "def main(req):\n  print(req)\n  return req"
			Expect(worker.SendFunction(function)).To(Succeed())
			Eventually(stdout).Should(gbytes.Say("{\"data\":true,\"type\":\"function_loaded\"}"))

			request := "jsonstring"
			response, err := worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal(request))
		})
	})
})

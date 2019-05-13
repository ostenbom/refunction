package worker_test

import (
	"io"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	. "github.com/ostenbom/refunction/worker"
)

var _ = XDescribe("Java Serverless Function Management", func() {
	var id string
	runtime := "java"
	var targetLayer string
	var worker *Worker
	var stdout *gbytes.Buffer
	var straceBuffer *gbytes.Buffer

	var echoFunction = "yv66vgAAADQAIQoABgAPCQAQABEKABIAEwoAFAAVBwAWBwAXAQAGPGluaXQ+AQADKClWAQAEQ29kZQEAD0xpbmVOdW1iZXJUYWJsZQEABmhhbmRsZQEALChMb3JnL2pzb24vSlNPTk9iamVjdDspTG9yZy9qc29uL0pTT05PYmplY3Q7AQAKU291cmNlRmlsZQEADUZ1bmN0aW9uLmphdmEMAAcACAcAGAwAGQAaBwAbDAAcAB0HAB4MAB8AIAEACEZ1bmN0aW9uAQAQamF2YS9sYW5nL09iamVjdAEAEGphdmEvbGFuZy9TeXN0ZW0BAANvdXQBABVMamF2YS9pby9QcmludFN0cmVhbTsBABNvcmcvanNvbi9KU09OT2JqZWN0AQAIdG9TdHJpbmcBABQoKUxqYXZhL2xhbmcvU3RyaW5nOwEAE2phdmEvaW8vUHJpbnRTdHJlYW0BAAdwcmludGxuAQAVKExqYXZhL2xhbmcvU3RyaW5nOylWACEABQAGAAAAAAACAAEABwAIAAEACQAAAB0AAQABAAAABSq3AAGxAAAAAQAKAAAABgABAAAAAwABAAsADAABAAkAAAAoAAIAAgAAAAyyAAIrtgADtgAEK7AAAAABAAoAAAAKAAIAAAAFAAoABgABAA0AAAACAA4="

	var yoloSwagFunction = "yv66vgAAADQAKgoACwAUCQAVABYKAAUAFwoAGAAZBwAaCgAFABQIABsIABwKAAUAHQcAHgcAHwEABjxpbml0PgEAAygpVgEABENvZGUBAA9MaW5lTnVtYmVyVGFibGUBAAZoYW5kbGUBACwoTG9yZy9qc29uL0pTT05PYmplY3Q7KUxvcmcvanNvbi9KU09OT2JqZWN0OwEAClNvdXJjZUZpbGUBAA1GdW5jdGlvbi5qYXZhDAAMAA0HACAMACEAIgwAIwAkBwAlDAAmACcBABNvcmcvanNvbi9KU09OT2JqZWN0AQAEeW9sbwEABHN3YWcMACgAKQEACEZ1bmN0aW9uAQAQamF2YS9sYW5nL09iamVjdAEAEGphdmEvbGFuZy9TeXN0ZW0BAANvdXQBABVMamF2YS9pby9QcmludFN0cmVhbTsBAAh0b1N0cmluZwEAFCgpTGphdmEvbGFuZy9TdHJpbmc7AQATamF2YS9pby9QcmludFN0cmVhbQEAB3ByaW50bG4BABUoTGphdmEvbGFuZy9TdHJpbmc7KVYBAANwdXQBADsoTGphdmEvbGFuZy9TdHJpbmc7TGphdmEvbGFuZy9PYmplY3Q7KUxvcmcvanNvbi9KU09OT2JqZWN0OwAhAAoACwAAAAAAAgABAAwADQABAA4AAAAdAAEAAQAAAAUqtwABsQAAAAEADwAAAAYAAQAAAAMAAQAQABEAAQAOAAAANQADAAIAAAAZsgACK7YAA7YABLsABVm3AAYSBxIItgAJsAAAAAEADwAAAAoAAgAAAAUACgAGAAEAEgAAAAIAEw=="

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
			Eventually(stdout).Should(gbytes.Say("{\"data\":true,\"type\":\"function_loaded\"}"))
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
			Eventually(stdout).Should(gbytes.Say("{\"data\":true,\"type\":\"function_loaded\"}"))

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

		XIt("can restore and change function", func() {
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
			Eventually(stdout).Should(gbytes.Say("{\"type\": \"function_loaded\", \"data\": false}"))

			function = "def main(req):\n  print(req)\n  return req"
			Expect(worker.SendFunction(function)).To(Succeed())
			Eventually(stdout).Should(gbytes.Say("{\"type\": \"function_loaded\", \"data\": true}"))

			request := "jsonstring"
			response, err := worker.SendRequest(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(Equal(request))
		})
	})
})

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

	var echoFunction = "UEsDBBQACAgIAFB9vE4AAAAAAAAAAAAAAAAJAAQATUVUQS1JTkYv/soAAAMAUEsHCAAAAAACAAAAAAAAAFBLAwQUAAgICABQfbxOAAAAAAAAAAAAAAAAFAAAAE1FVEEtSU5GL01BTklGRVNULk1G803My0xLLS7RDUstKs7Mz7NSMNQz4OVyLkpNLElN0XWqBAlY6BnEG1oaKmj4FyUm56QqOOcXFeQXJZYA1WvycvFyAQBQSwcIWeqwU0QAAABFAAAAUEsDBBQACAgIAOJzvE4AAAAAAAAAAAAAAAAOAAAARnVuY3Rpb24uY2xhc3N9UclOwzAQfe7mNqS00JalUKC3pAdy4QSIC1IPKAKkot7TYEWuEhulCRJ/BRyKxIEP4KMQk7AIxOLDjGfmvTcz9vPL4xOAPfQNVNCooYklA8toGWijw7HCscpQOZRKJkcMRcseM5SO9aVgaLhSidM0moj4wpuElClFnlQM+5br68gJtA5C4QQzrZwTMmeTqfCTA/u/IoMx0mnsi6HMBOvDVPmJ1Gp36l17JjiqHGsm1tHl2DCxiR7Hlolt7DBUP8AMzQzuhJ4KnDfhb6nRzSwREW2jUyp03LwitXMeS5WMklh4EQ3S/XtM6pVoAkoVMLQt2/2inWeJ3vpFlYFfZVFII3asnyx7jD7K9BHZKYBl+5KtUdQjz8iXBw9gd3ShlyJbyZNFgizAfIdaORUw71EYzFGcozS4/WQYVCUZGDm3njdafAVQSwcIfQ4BzEQBAAAHAgAAUEsBAhQAFAAICAgAUH28TgAAAAACAAAAAAAAAAkABAAAAAAAAAAAAAAAAAAAAE1FVEEtSU5GL/7KAABQSwECFAAUAAgICABQfbxOWeqwU0QAAABFAAAAFAAAAAAAAAAAAAAAAAA9AAAATUVUQS1JTkYvTUFOSUZFU1QuTUZQSwECFAAUAAgI"

	var yoloSwagFunction = "UEsDBBQACAgIAFeCvE4AAAAAAAAAAAAAAAAJAAQATUVUQS1JTkYv/soAAAMAUEsHCAAAAAACAAAAAAAAAFBLAwQUAAgICABXgrxOAAAAAAAAAAAAAAAAFAAAAE1FVEEtSU5GL01BTklGRVNULk1G803My0xLLS7RDUstKs7Mz7NSMNQz4OVyLkpNLElN0XWqBAlY6BnEG1oaKmj4FyUm56QqOOcXFeQXJZYA1WvycvFyAQBQSwcIWeqwU0QAAABFAAAAUEsDBBQACAgIAE6CvE4AAAAAAAAAAAAAAAAOAAAARnVuY3Rpb24uY2xhc3N9UtlKw0AUPdMtaY1bbV1q3be0gnnxSUUEoQ8StFARfEzbIUxJZiRNlf6V+lBBwQ/wo8Q7RUWwOg93P2fu3Dtv78+vAPZRzWEMhSyKmM0hjbkc5rFgoKSdgolFE2VtLhlYNrDCkDkSUsTHDEm7csWQOlVtzjDpCsnPe2GTR5deM6BIKvSEZDiw3ZYKHV8pP+CO31XSOSNx0ezwVnxY+S/JkGuoXtTiNaEJx2s92YqFknsd79azYGHcwKqFNaxb2MCmgS0L29hhKP1NSn31VaBIde8834KNCoP5xcwwpbmdwJO+8wX4EWr0uzEP6emqR4miO8wI5dQjIeNGHHEvpK7NWJEtpM9QsCvuD/gwShUzI4AMxo32AuqiaP9G6WGPee12PVI3PIr7DDsjqkbhaEJ6x/okwPTcSE6Qt0SakU5Xn8AeyKA9kswMg0lkMYXpz9IT8hOky49IVAdIDpB6Qfr6CRl3N2/kzQGyu/ffDHmkNC1dmyGbPhNMyuSHDcx8AFBLBwhlD2a4fwEAAHoCAABQSwECFAAUAAgICABXgrxOAAAAAAIAAAAAAAAACQAEAAAAAAAAAAAAAAAAAAAATUVUQS1JTkYv/soAAFBLAQIUABQACAgIAFeCvE5Z6rBTRAAAAEUAAAAUAAAAAAAAAAAAAAAAAD0AAABNRVRBLUlORi9NQU5JRkVTVC5NRlBLAQIUABQACAgIAE6CvE5lD2a4fwEAAHoCAAAOAAAAAAAAAAAAAAAAAMMAAABGdW5jdGlvbi5jbGFzc1BLBQYAAAAAAwADALkAAAB+AgAAAAA="

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

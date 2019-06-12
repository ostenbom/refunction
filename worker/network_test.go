package worker_test

import (
	"io/ioutil"
	"net"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	. "github.com/ostenbom/refunction/worker"
)

var _ = Describe("Network", func() {

	var id string

	BeforeEach(func() {
		id = strconv.Itoa(GinkgoParallelNode())
	})

	Describe("when reaching to external internet", func() {
		var worker *Worker
		var targetLayer string
		var stdout *gbytes.Buffer
		var stderr *gbytes.Buffer
		runtime := "python"

		JustBeforeEach(func() {
			var err error
			worker, err = NewWorker(id, client, runtime, targetLayer)
			Expect(err).NotTo(HaveOccurred())
			stdout = gbytes.NewBuffer()
			stderr = gbytes.NewBuffer()
			worker.WithStdPipes(stderr, stdout, GinkgoWriter)
			Expect(worker.Start()).To(Succeed())
		})

		AfterEach(func() {
			err := worker.End()
			Expect(err).NotTo(HaveOccurred())
		})

		BeforeEach(func() {
			targetLayer = "ping-google.py"
		})

		It("can get a response", func() {
			Skip("Does not work on doc vms")
			Eventually(stdout).Should(gbytes.Say("ttl="))
		})
	})

	Describe("communicating with a container", func() {
		var worker *Worker
		var targetLayer string
		runtime := "python"

		JustBeforeEach(func() {
			var err error
			worker, err = NewWorker(id, client, runtime, targetLayer)
			Expect(err).NotTo(HaveOccurred())
			Expect(worker.Start()).To(Succeed())
		})

		AfterEach(func() {
			err := worker.End()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when we are listening and responding on a port", func() {
			BeforeEach(func() {
				targetLayer = "tcp-server.py"
			})

			It("has an ip", func() {
				Expect(worker.IP).NotTo(BeNil())
			})

			// Flakey test. Functionality currently not in use since communication done
			// via stdin
			XIt("can connect and echo", func() {
				tcpAddr := net.TCPAddr{
					IP:   worker.IP,
					Port: 5000,
				}

				conn, err := net.DialTCP("tcp", nil, &tcpAddr)
				defer conn.Close()
				Expect(err).NotTo(HaveOccurred())

				writeBytes := []byte("hello there!\n")
				_, err = conn.Write(writeBytes)
				Expect(err).NotTo(HaveOccurred())

				result, err := ioutil.ReadAll(conn)
				Expect(err).NotTo(HaveOccurred())

				Expect(result).To(Equal(writeBytes))
			})
		})
	})
})

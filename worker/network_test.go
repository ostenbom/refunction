package worker_test

import (
	"io/ioutil"
	"net"
	"strconv"

	"github.com/containerd/containerd/cio"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/ostenbom/refunction/worker"
)

var _ = Describe("Network", func() {

	var id string

	BeforeEach(func() {
		id = strconv.Itoa(GinkgoParallelNode())
	})

	Describe("communicating with a container", func() {
		var worker *Worker
		var targetLayer string
		runtime := "python"

		JustBeforeEach(func() {
			var err error
			worker, err = NewWorker(id, client, runtime, targetLayer)
			worker.WithCreator(cio.NewCreator(cio.WithStreams(nil, GinkgoWriter, GinkgoWriter)))
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

			It("can connect and echo", func() {
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

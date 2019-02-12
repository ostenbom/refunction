package worker_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/ostenbom/refunction/worker"
)

var _ = Describe("Worker Restoring", func() {
	var id string

	BeforeEach(func() {
		id = strconv.Itoa(GinkgoParallelNode())
	})

	Describe("state restoring", func() {
		var worker *Worker
		var targetLayer string
		var runtime string

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

		Context("when the program changes stack variables", func() {
			BeforeEach(func() {
				runtime = "alpine"
				targetLayer = "forloopfile"
			})

			It("can restore dirty stack pages", func() {
				countLocation := getRootfs(worker) + "count.txt"

				Eventually(func() bool {
					_, err := os.Stat(countLocation)
					return os.IsNotExist(err)
				}).Should(BeFalse())

				Expect(worker.Attach()).To(Succeed())
				defer worker.Detach()

				// Work out what will be printed next
				countContent, err := ioutil.ReadFile(countLocation)
				Expect(err).NotTo(HaveOccurred())
				lines := strings.Split(string(countContent), "\n")
				lastLine := lines[len(lines)-2]
				lastLineItems := strings.Split(lastLine, " ")
				number, err := strconv.Atoi(lastLineItems[len(lastLineItems)-1])
				Expect(err).NotTo(HaveOccurred())
				incrementedLine := fmt.Sprintf("at: %d", number+1)

				// Get first state
				state, err := worker.GetState()
				Expect(err).NotTo(HaveOccurred())
				Expect(state.SavePages("[stack]")).To(Succeed())
				Expect(worker.ClearMemRefs()).To(Succeed())
				Expect(worker.Continue()).To(Succeed())

				// Let run, restore
				time.Sleep(time.Millisecond * 60)
				Expect(worker.Stop()).To(Succeed())
				err = state.RestoreDirtyPages("[stack]")
				Expect(err).NotTo(HaveOccurred())
				Expect(worker.Continue()).To(Succeed())

				// Let run, check variable was restored
				time.Sleep(time.Millisecond * 60)
				Expect(worker.Stop()).To(Succeed())
				countContent, err = ioutil.ReadFile(countLocation)
				Expect(err).NotTo(HaveOccurred())
				numberPrintedIncrements := strings.Count(string(countContent), incrementedLine)
				Expect(numberPrintedIncrements).To(Equal(2))
			})
		})

		Context("when the program changes register variables", func() {
			BeforeEach(func() {
				runtime = "alpine"
				targetLayer = "forloopregisterfile"
			})

			It("can restore variable in a register", func() {
				countLocation := getRootfs(worker) + "count.txt"

				Eventually(func() bool {
					_, err := os.Stat(countLocation)
					return os.IsNotExist(err)
				}).Should(BeFalse())

				Expect(worker.Attach()).To(Succeed())
				defer worker.Detach()

				// Work out what will be printed next
				countContent, err := ioutil.ReadFile(countLocation)
				Expect(err).NotTo(HaveOccurred())
				lines := strings.Split(string(countContent), "\n")
				lastLine := lines[len(lines)-2]
				lastLineItems := strings.Split(lastLine, " ")
				number, err := strconv.Atoi(lastLineItems[len(lastLineItems)-1])
				Expect(err).NotTo(HaveOccurred())
				incrementedLine := fmt.Sprintf("at: %d", number+1)

				// Get first state
				state, err := worker.GetState()
				Expect(err).NotTo(HaveOccurred())
				Expect(worker.Continue()).To(Succeed())

				// Let run, restore
				time.Sleep(time.Millisecond * 60)
				err = worker.SetRegs(state)
				Expect(err).NotTo(HaveOccurred())

				// Let run, check variable was restored
				time.Sleep(time.Millisecond * 60)
				Expect(worker.Stop()).To(Succeed())
				countContent, err = ioutil.ReadFile(countLocation)
				Expect(err).NotTo(HaveOccurred())
				numberPrintedIncrements := strings.Count(string(countContent), incrementedLine)
				Expect(numberPrintedIncrements).To(Equal(2))
			})
		})

		Context("when a python program changes stack variables", func() {
			BeforeEach(func() {
				runtime = "python"
				targetLayer = "forloop.py"
			})

			It("can restore a for loop variable", func() {
				// Initiate python ready sequence
				Expect(worker.Attach()).To(Succeed())
				defer worker.Detach()
				Expect(worker.Continue()).To(Succeed())
				Expect(worker.AwaitSignal()).To(Succeed())
				Expect(worker.SendEnableSignal()).To(Succeed())

				countLocation := getRootfs(worker) + "/tmp/count.txt"

				Eventually(func() bool {
					_, err := os.Stat(countLocation)
					return os.IsNotExist(err)
				}).Should(BeFalse())

				time.Sleep(time.Millisecond * 50)
				Expect(worker.Stop()).To(Succeed())

				// Work out what will be printed next
				countContent, err := ioutil.ReadFile(countLocation)
				Expect(err).NotTo(HaveOccurred())
				lines := strings.Split(string(countContent), "\n")
				lastLine := lines[len(lines)-2]
				lastLineItems := strings.Split(lastLine, " ")
				number, err := strconv.Atoi(lastLineItems[len(lastLineItems)-1])
				Expect(err).NotTo(HaveOccurred())
				incrementedLine := fmt.Sprintf("at: %d", number+1)

				// Get first state
				state, err := worker.GetState()
				Expect(err).NotTo(HaveOccurred())
				Expect(state.SavePages("[stack]")).To(Succeed())
				Expect(state.SavePages("[heap]")).To(Succeed())
				Expect(worker.ClearMemRefs()).To(Succeed())
				Expect(worker.Continue()).To(Succeed())

				// Let run, restore
				time.Sleep(time.Millisecond * 60)
				Expect(worker.Stop()).To(Succeed())
				start := time.Now()
				Expect(state.RestoreDirtyPages("[stack]"))
				Expect(state.RestoreDirtyPages("[heap]"))
				Expect(state.RestoreRegs()).To(Succeed())
				fmt.Printf("restore time: %s\n", time.Since(start))
				Expect(worker.Continue()).To(Succeed())

				// Let run, check variable was restored
				time.Sleep(time.Millisecond * 60)
				Expect(worker.Stop()).To(Succeed())
				countContent, err = ioutil.ReadFile(countLocation)
				Expect(err).NotTo(HaveOccurred())
				numberPrintedIncrements := strings.Count(string(countContent), incrementedLine)
				Expect(numberPrintedIncrements).To(Equal(2))
			})
		})
	})
})

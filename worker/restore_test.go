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

	Describe("for loop file", func() {
		var worker *Worker
		var targetLayer string

		BeforeEach(func() {
			var err error
			targetLayer = "forloopfile"

			worker, err = NewWorker(id, client, targetLayer)
			Expect(err).NotTo(HaveOccurred())
			Expect(worker.Start()).To(Succeed())
		})

		AfterEach(func() {
			err := worker.End()
			Expect(err).NotTo(HaveOccurred())
		})

		It("can restore dirty stack pages", func() {
			countLocation := getRootfs(worker, targetLayer) + "count.txt"

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

		It("can reset the stack variable", func() {

		})
	})
})

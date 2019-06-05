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
			worker.WithSyscallTrace(GinkgoWriter)
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
				WaitFileExists(countLocation)

				Expect(worker.Attach()).To(Succeed())
				Expect(worker.Stop()).To(Succeed())
				defer worker.Detach()

				// Work out what will be printed next
				incrementedLine := CalculateNextCountLine(countLocation)

				// Get first state
				state, err := worker.GetState()
				Expect(err).NotTo(HaveOccurred())
				Expect(state.SaveWritablePages()).To(Succeed())
				Expect(worker.ClearMemRefs()).To(Succeed())
				worker.Continue()

				// Let run, restore
				time.Sleep(time.Millisecond * 60)
				Expect(worker.Stop()).To(Succeed())
				err = state.RestoreDirtyPages()
				Expect(err).NotTo(HaveOccurred())
				worker.Continue()

				// Let run, check variable was restored
				time.Sleep(time.Millisecond * 60)
				Expect(worker.Stop()).To(Succeed())
				countContent, err := ioutil.ReadFile(countLocation)
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
				WaitFileExists(countLocation)

				Expect(worker.Attach()).To(Succeed())
				Expect(worker.Stop()).To(Succeed())
				defer worker.Detach()

				// Work out what will be printed next
				incrementedLine := CalculateNextCountLine(countLocation)

				// Get first state
				state, err := worker.GetState()
				Expect(err).NotTo(HaveOccurred())
				worker.Continue()

				// Let run, restore
				time.Sleep(time.Millisecond * 60)
				Expect(worker.Stop()).To(Succeed())
				err = worker.SetRegs(state)
				Expect(err).NotTo(HaveOccurred())
				worker.Continue()

				// Let run, check variable was restored
				time.Sleep(time.Millisecond * 60)
				Expect(worker.Stop()).To(Succeed())
				countContent, err := ioutil.ReadFile(countLocation)
				Expect(err).NotTo(HaveOccurred())
				numberPrintedIncrements := strings.Count(string(countContent), incrementedLine)
				Expect(numberPrintedIncrements).To(Equal(2))
			})
		})

		Context("when the program changes initialized variables", func() {
			BeforeEach(func() {
				runtime = "alpine"
				targetLayer = "forloopinitializedvar"
			})

			It("can restore the variable", func() {
				countLocation := getRootfs(worker) + "count.txt"
				WaitFileExists(countLocation)

				Expect(worker.Attach()).To(Succeed())
				Expect(worker.Stop()).To(Succeed())
				defer worker.Detach()

				// Work out what will be printed next
				incrementedLine := CalculateNextCountLine(countLocation)

				// Get first state
				state, err := worker.GetState()
				Expect(err).NotTo(HaveOccurred())
				Expect(state.SaveWritablePages()).To(Succeed())
				Expect(worker.ClearMemRefs()).To(Succeed())
				worker.Continue()

				// Let run, restore
				time.Sleep(time.Millisecond * 60)
				Expect(worker.Stop()).To(Succeed())
				err = state.RestoreDirtyPages()
				Expect(err).NotTo(HaveOccurred())
				worker.Continue()

				// Let run, check variable was restored
				time.Sleep(time.Millisecond * 60)
				Expect(worker.Stop()).To(Succeed())
				countContent, err := ioutil.ReadFile(countLocation)
				Expect(err).NotTo(HaveOccurred())
				numberPrintedIncrements := strings.Count(string(countContent), incrementedLine)
				Expect(numberPrintedIncrements).To(Equal(2))
			})
		})

		Context("when the program changes uninitialized variables", func() {
			BeforeEach(func() {
				runtime = "alpine"
				targetLayer = "forloopuninitializedvar"
			})

			It("can restore the variable", func() {
				countLocation := getRootfs(worker) + "count.txt"
				WaitFileExists(countLocation)

				Expect(worker.Attach()).To(Succeed())
				Expect(worker.Stop()).To(Succeed())
				defer worker.Detach()

				// Work out what will be printed next
				incrementedLine := CalculateNextCountLine(countLocation)

				// Get first state
				state, err := worker.GetState()
				Expect(err).NotTo(HaveOccurred())
				Expect(state.SaveWritablePages()).To(Succeed())
				Expect(worker.ClearMemRefs()).To(Succeed())
				worker.Continue()

				// Let run, restore
				time.Sleep(time.Millisecond * 60)
				Expect(worker.Stop()).To(Succeed())
				err = state.RestoreDirtyPages()
				Expect(err).NotTo(HaveOccurred())
				worker.Continue()

				// Let run, check variable was restored
				time.Sleep(time.Millisecond * 60)
				Expect(worker.Stop()).To(Succeed())
				countContent, err := ioutil.ReadFile(countLocation)
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

				countLocation := getRootfs(worker) + "/tmp/count.txt"
				WaitFileExists(countLocation)

				time.Sleep(time.Millisecond * 60)
				Expect(worker.Stop()).To(Succeed())

				// Work out what will be printed next
				incrementedLine := CalculateNextCountLine(countLocation)

				// Get first state
				state, err := worker.GetState()
				Expect(err).NotTo(HaveOccurred())
				Expect(state.SaveWritablePages()).To(Succeed())
				Expect(worker.ClearMemRefs()).To(Succeed())
				worker.Continue()

				// Let run, restore
				time.Sleep(time.Millisecond * 60)
				Expect(worker.Stop()).To(Succeed())
				start := time.Now()
				Expect(state.RestoreDirtyPages()).To(Succeed())
				Expect(state.RestoreRegs()).To(Succeed())
				fmt.Printf("restore time: %s\n", time.Since(start))
				worker.Continue()

				// Let run, check variable was restored
				time.Sleep(time.Millisecond * 60)
				Expect(worker.Stop()).To(Succeed())
				countContent, err := ioutil.ReadFile(countLocation)
				Expect(err).NotTo(HaveOccurred())
				numberPrintedIncrements := strings.Count(string(countContent), incrementedLine)
				Expect(numberPrintedIncrements).To(Equal(2))
			})
		})

		Context("when a python program runs multiple threads", func() {
			BeforeEach(func() {
				runtime = "python"
				targetLayer = "threads.py"
			})

			It("can restore both threads", func() {
				countLocation := getRootfs(worker) + "/tmp/count.txt"
				WaitFileExists(countLocation)
				// Initiate python ready sequence
				Expect(worker.Attach()).To(Succeed())
				defer worker.Detach()

				time.Sleep(time.Millisecond * 60)
				Expect(worker.Stop()).To(Succeed())

				// Work out what will be printed next
				incrementedLine := CalculateNextCountLine(countLocation)

				// Get first state
				state, err := worker.GetState()
				Expect(err).NotTo(HaveOccurred())
				Expect(state.SaveWritablePages()).To(Succeed())
				Expect(worker.ClearMemRefs()).To(Succeed())

				worker.Continue()

				// Let run, restore
				time.Sleep(time.Millisecond * 60)
				Expect(worker.Stop()).To(Succeed())
				start := time.Now()
				Expect(state.RestoreDirtyPages()).To(Succeed())
				Expect(state.RestoreRegs()).To(Succeed())
				fmt.Printf("restore time: %s\n", time.Since(start))

				// Let run, check variable was restored
				worker.Continue()
				time.Sleep(time.Millisecond * 60)
				Expect(worker.Stop()).To(Succeed())
				countContent, err := ioutil.ReadFile(countLocation)
				Expect(err).NotTo(HaveOccurred())
				numberPrintedIncrements := strings.Count(string(countContent), incrementedLine)
				Expect(numberPrintedIncrements).To(Equal(2))
			})
		})

		Context("when a python program uses enough memory for the program break to change", func() {
			largeMemoryFunc := `
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

			BeforeEach(func() {
				runtime = "python"
				targetLayer = "serverless-function.py"
			})

			It("can see if the program break has changed", func() {
				Expect(worker.Activate()).To(Succeed())
				Expect(worker.SendFunction(largeMemoryFunc)).To(Succeed())
				response, err := worker.SendRequest("")
				Expect(err).NotTo(HaveOccurred())

				Expect(len(response.(string))).To(Equal(100000))

				state, err := worker.GetInitialCheckpoint()
				Expect(err).NotTo(HaveOccurred())

				Expect(state.MemoryChanged()).To(BeTrue())
				Expect(state.ProgramBreakChanged()).To(BeTrue())
			})

			It("can reset the program break", func() {
				Expect(worker.Activate()).To(Succeed())
				initialState, err := worker.GetInitialCheckpoint()
				Expect(err).NotTo(HaveOccurred())

				Expect(worker.SendFunction(largeMemoryFunc)).To(Succeed())
				response, err := worker.SendRequest("")
				Expect(err).NotTo(HaveOccurred())

				Expect(len(response.(string))).To(Equal(100000))

				Expect(worker.Restore()).To(Succeed())

				changed, err := initialState.ProgramBreakChanged()
				Expect(err).NotTo(HaveOccurred())
				Expect(changed).To(BeFalse())
			})

			It("can remove new memory areas", func() {
				Expect(worker.Activate()).To(Succeed())
				initialState, err := worker.GetInitialCheckpoint()
				Expect(err).NotTo(HaveOccurred())

				Expect(worker.SendFunction(largeMemoryFunc)).To(Succeed())
				response, err := worker.SendRequest("")
				Expect(err).NotTo(HaveOccurred())

				Expect(len(response.(string))).To(Equal(100000))

				changed, err := initialState.NumMemoryLocationsChanged()
				Expect(err).NotTo(HaveOccurred())
				Expect(changed).To(BeTrue())

				Expect(worker.Restore()).To(Succeed())

				changed, err = initialState.NumMemoryLocationsChanged()
				Expect(err).NotTo(HaveOccurred())
				Expect(changed).To(BeFalse())
			})

			// TODO: We are not testing for mremaps here
			It("leaves all memory the same as it was after restore", func() {
				Expect(worker.Activate()).To(Succeed())
				initialState, err := worker.GetInitialCheckpoint()
				Expect(err).NotTo(HaveOccurred())

				Expect(worker.SendFunction(largeMemoryFunc)).To(Succeed())
				response, err := worker.SendRequest("")
				Expect(err).NotTo(HaveOccurred())

				Expect(len(response.(string))).To(Equal(100000))

				changed, err := initialState.MemoryChanged()
				Expect(err).NotTo(HaveOccurred())
				Expect(changed).To(BeTrue())

				Expect(worker.Restore()).To(Succeed())

				changed, err = initialState.MemoryChanged()
				Expect(err).NotTo(HaveOccurred())
				Expect(changed).To(BeFalse())
			})
		})
	})
})

func WaitFileExists(location string) {
	Eventually(func() bool {
		_, err := os.Stat(location)
		return os.IsNotExist(err)
	}).Should(BeFalse())
}

func CalculateNextCountLine(countLocation string) string {
	countContent, err := ioutil.ReadFile(countLocation)
	Expect(err).NotTo(HaveOccurred())
	lines := strings.Split(string(countContent), "\n")
	lastLine := lines[len(lines)-2]
	lastLineItems := strings.Split(lastLine, " ")
	number, err := strconv.Atoi(lastLineItems[len(lastLineItems)-1])
	Expect(err).NotTo(HaveOccurred())
	return fmt.Sprintf("at: %d", number+1)
}

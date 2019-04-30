package worker_test

import (
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/sys/unix"

	. "github.com/ostenbom/refunction/worker"
)

var _ = Describe("Worker Manager checkpointing", func() {
	var id string
	var worker *Worker
	var targetLayer string
	runtime := "alpine"

	BeforeEach(func() {
		id = strconv.Itoa(GinkgoParallelNode())
	})

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

	Describe("rlimit checkpointing", func() {
		BeforeEach(func() {
			targetLayer = "forloopstack"
		})

		It("can get the processes limits", func() {
			Expect(worker.Attach()).To(Succeed())
			state, err := worker.GetState()
			Expect(err).NotTo(HaveOccurred())

			rlimits := state.GetRlimits()
			_, exists := rlimits[unix.RLIMIT_AS]
			Expect(exists).To(BeTrue())
			_, exists = rlimits[unix.RLIMIT_DATA]
			Expect(exists).To(BeTrue())
			_, exists = rlimits[unix.RLIMIT_STACK]
			Expect(exists).To(BeTrue())
		})
	})

	Describe("memory checkpointing", func() {

		Context("for loop stack + heap", func() {
			BeforeEach(func() {
				targetLayer = "forloopheap"
			})

			It("can clear memory refs", func() {
				Expect(worker.Attach()).To(Succeed())
				Expect(worker.ClearMemRefs()).To(Succeed())

				state, err := worker.GetState()
				Expect(err).NotTo(HaveOccurred())

				// while still stopped, we expect there to be no dirty pages
				dirtyStack, err := state.CountDirtyPages("[stack]")
				dirtyHeap, err2 := state.CountDirtyPages("[heap]")
				Expect(worker.Detach()).To(Succeed())

				Expect(err).NotTo(HaveOccurred())
				Expect(dirtyStack).To(Equal(0))
				Expect(err2).NotTo(HaveOccurred())
				Expect(dirtyHeap).To(Equal(0))
			})

			It("knows when the heap has been modified", func() {
				Expect(worker.Attach()).To(Succeed())
				Expect(worker.ClearMemRefs()).To(Succeed())
				worker.Continue()

				// mallocs every 50ms
				time.Sleep(time.Millisecond * 60)
				Expect(worker.Stop()).To(Succeed())
				state, err := worker.GetState()
				Expect(err).NotTo(HaveOccurred())

				// after a bit, we expect the heap to change
				dirtyHeap, err := state.CountDirtyPages("[heap]")
				Expect(worker.Detach()).To(Succeed())

				Expect(err).NotTo(HaveOccurred())
				Expect(dirtyHeap).NotTo(Equal(0))
			})

		})

		Context("for loop stack", func() {
			BeforeEach(func() {
				targetLayer = "forloopstack"
			})

			It("can notice a variable change the stack", func() {
				Expect(worker.Attach()).To(Succeed())
				Expect(worker.ClearMemRefs()).To(Succeed())
				worker.Continue()

				// loop ticks every 50ms
				time.Sleep(time.Millisecond * 60)
				state, err := worker.GetState()
				Expect(err).NotTo(HaveOccurred())

				dirtyStack, err := state.CountDirtyPages("[stack]")
				Expect(worker.Detach()).To(Succeed())

				Expect(err).NotTo(HaveOccurred())
				Expect(dirtyStack).NotTo(Equal(0))
			})

			It("can make a copy of an area of memory", func() {
				// loop ticks every 50ms
				time.Sleep(time.Millisecond * 60)
				Expect(worker.Attach()).To(Succeed())
				defer worker.Detach()

				state, err := worker.GetState()
				Expect(err).NotTo(HaveOccurred())

				Expect(state.SaveWritablePages()).To(Succeed())

				memSize := state.MemorySize()
				Expect(memSize).NotTo(Equal(0))
			})

			It("has no memory region changes", func() {
				Expect(worker.Attach()).To(Succeed())
				defer worker.Detach()

				state, err := worker.GetState()
				Expect(err).NotTo(HaveOccurred())
				worker.Continue()

				time.Sleep(time.Millisecond * 100)
				Expect(worker.Stop()).To(Succeed())
				changed, err := state.MemoryChanged()
				Expect(err).NotTo(HaveOccurred())
				Expect(changed).To(BeFalse())
			})

			It("has three file descriptors", func() {
				Expect(worker.Attach()).To(Succeed())
				defer worker.Detach()

				state, err := worker.GetState()
				Expect(err).NotTo(HaveOccurred())
				worker.Continue()

				Expect(len(state.GetFileDescriptors())).To(Equal(3))
			})

			It("has no changes in files", func() {
				Expect(worker.Attach()).To(Succeed())
				defer worker.Detach()

				state, err := worker.GetState()
				Expect(err).NotTo(HaveOccurred())
				worker.Continue()

				time.Sleep(time.Millisecond * 60)
				Expect(worker.Stop()).To(Succeed())
				changed, err := state.FdsChanged()
				Expect(err).NotTo(HaveOccurred())
				Expect(changed).To(BeFalse())
			})

			It("has a program counter", func() {
				Expect(worker.Attach()).To(Succeed())
				defer worker.Detach()

				state, err := worker.GetState()
				Expect(err).NotTo(HaveOccurred())
				Expect(state.PC()).NotTo(Equal(0))
			})
		})

		Context("expanding heap", func() {
			BeforeEach(func() {
				targetLayer = "growingheap"
			})

			It("notices when the process changes memory regions", func() {
				Expect(worker.Attach()).To(Succeed())
				defer worker.Detach()

				state, err := worker.GetState()
				Expect(err).NotTo(HaveOccurred())
				worker.Continue()

				time.Sleep(time.Millisecond * 100)
				Expect(worker.Stop()).To(Succeed())
				changed, err := state.MemoryChanged()
				Expect(err).NotTo(HaveOccurred())
				Expect(changed).To(BeTrue())
			})
		})

		Context("opened files", func() {
			BeforeEach(func() {
				targetLayer = "fileopener"
			})

			It("notices changes in files", func() {
				Expect(worker.Attach()).To(Succeed())
				defer worker.Detach()

				state, err := worker.GetState()
				Expect(err).NotTo(HaveOccurred())
				worker.Continue()

				time.Sleep(time.Millisecond * 60)
				Expect(worker.Stop()).To(Succeed())
				changed, err := state.FdsChanged()
				Expect(err).NotTo(HaveOccurred())
				Expect(changed).To(BeTrue())
			})
		})
	})
})

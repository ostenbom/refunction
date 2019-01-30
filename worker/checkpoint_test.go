package worker_test

import (
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/ostenbom/refunction/worker"
)

var _ = Describe("Worker Manager checkpointing", func() {
	var id string

	BeforeEach(func() {
		id = strconv.Itoa(GinkgoParallelNode())
	})

	Describe("for loop stack + heap", func() {
		var worker *Worker
		var targetLayer string

		BeforeEach(func() {
			var err error
			targetLayer = "forloopheap"
			worker, err = NewWorker(id, client, targetLayer)
			Expect(err).NotTo(HaveOccurred())

			Expect(worker.Start()).To(Succeed())
		})

		AfterEach(func() {
			err := worker.End()
			Expect(err).NotTo(HaveOccurred())
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
			Expect(worker.Continue()).To(Succeed())

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

	Describe("for loop stack", func() {
		var worker *Worker
		var targetLayer string

		BeforeEach(func() {
			var err error
			targetLayer = "forloopstack"
			worker, err = NewWorker(id, client, targetLayer)
			Expect(err).NotTo(HaveOccurred())

			Expect(worker.Start()).To(Succeed())
		})

		AfterEach(func() {
			err := worker.End()
			Expect(err).NotTo(HaveOccurred())
		})

		It("can notice a variable change the stack", func() {
			Expect(worker.Attach()).To(Succeed())
			Expect(worker.ClearMemRefs()).To(Succeed())
			Expect(worker.Continue()).To(Succeed())

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

			Expect(state.SavePages("[stack]")).To(Succeed())

			memSize, err := state.MemorySize("[stack]")
			Expect(err).NotTo(HaveOccurred())
			Expect(memSize).NotTo(Equal(0))
		})
	})

})

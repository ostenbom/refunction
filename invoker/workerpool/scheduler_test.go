package workerpool_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/ostenbom/refunction/invoker/workerpool"
)

var _ = Describe("Scheduler", func() {
	var (
		scheduler        *Scheduler
		w1               *ScheduleWorker
		w2               *ScheduleWorker
		decommissionTime time.Duration
	)

	BeforeEach(func() {
		w1 = &ScheduleWorker{}
		w2 = &ScheduleWorker{}
		workers := make(map[string]*ScheduleWorker)
		workers["one"] = w1
		workers["two"] = w2
		decommissionTime = time.Millisecond * 20
		scheduler = NewFakeScheduler(workers, []string{"one", "two"}, decommissionTime)
	})

	Describe("RunUndeployed", func() {
		It("moves something from undeployed to running", func() {
			name, _ := scheduler.RunUndeployed()
			IsIn(scheduler, name, false, false, true)
		})
	})

	Describe("RunComplete", func() {
		It("moves it from running to deployed", func() {
			name, _ := scheduler.RunUndeployed()
			scheduler.RunComplete(name)
			IsIn(scheduler, name, false, true, false)
		})
	})

	Describe("RunDeployedFunction", func() {
		It("returns false if no deployed function exists", func() {
			_, _, exists := scheduler.RunDeployedFunction("one")
			Expect(exists).To(BeFalse())
		})

		It("returns true if a deployed function exists", func() {
			name, sw := scheduler.RunUndeployed()
			sw.SetFunction("def func")
			scheduler.RunComplete(name)
			_, _, exists := scheduler.RunDeployedFunction("def func")
			Expect(exists).To(BeTrue())
		})

		It("returns the correct name", func() {
			name, sw := scheduler.RunUndeployed()
			sw.SetFunction("def func")
			scheduler.RunComplete(name)
			returnedName, _, _ := scheduler.RunDeployedFunction("def func")
			Expect(name).To(Equal(returnedName))
		})

		It("moves it from deployed to running", func() {
			name, sw := scheduler.RunUndeployed()
			sw.SetFunction("def func")
			scheduler.RunComplete(name)
			returnedName, _, _ := scheduler.RunDeployedFunction("def func")
			IsIn(scheduler, returnedName, false, false, true)
		})
	})

	Describe("ScheduleDecommission", func() {
		It("moves from deployed to undeployed after decomissionTime", func() {
			name, sw := scheduler.RunUndeployed()
			scheduler.RunComplete(name)
			scheduler.ScheduleDecommission(name, sw)
			time.Sleep(decommissionTime)
			// Some extra to do the move
			time.Sleep(time.Millisecond * 2)
			IsIn(scheduler, name, true, false, false)
		})

		It("does not move if the function is run again", func() {
			name, sw := scheduler.RunUndeployed()
			sw.SetFunction("def func")
			scheduler.RunComplete(name)
			scheduler.ScheduleDecommission(name, sw)
			time.Sleep(decommissionTime / 2)
			_, sw, exists := scheduler.RunDeployedFunction("def func")
			Expect(exists).To(BeTrue())
			sw.MarkRunTime()
			scheduler.RunComplete(name)
			time.Sleep(decommissionTime / 2)
			IsIn(scheduler, name, false, true, false)
		})
	})

})

func IsIn(s *Scheduler, name string, inUndeployed, inDeployed, inRunning bool) {
	undeployed := s.UndeployedWorkers()
	deployed := s.DeployedWorkers()
	running := s.RunningWorkers()
	Expect(Contains(name, undeployed)).To(Equal(inUndeployed))
	Expect(Contains(name, deployed)).To(Equal(inDeployed))
	Expect(Contains(name, running)).To(Equal(inRunning))
}

func Contains(target string, in []string) bool {
	for _, src := range in {
		if src == target {
			return true
		}
	}
	return false
}

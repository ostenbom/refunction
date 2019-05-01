package workerpool

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ostenbom/refunction/invoker/storage"
	"github.com/ostenbom/refunction/worker"
)

const decomissionTime = time.Second

type Scheduler struct {
	workers    map[string]*ScheduleWorker
	undeployed []string
	deployed   []string
	running    []string
	mux        sync.Mutex
}

type ScheduleWorker struct {
	worker   *worker.Worker
	runTime  time.Time
	function string
}

func NewScheduler(workers []*worker.Worker) *Scheduler {
	scheduleWorkers := make(map[string]*ScheduleWorker)
	var undeployed []string

	for _, w := range workers {
		scheduleWorkers[w.ID] = &ScheduleWorker{
			worker: w,
		}
		undeployed = append(undeployed, w.ID)
	}

	return &Scheduler{
		workers:    scheduleWorkers,
		undeployed: undeployed,
	}
}

func (p *WorkerPool) Run(function *storage.Function, request string) (string, error) {
	fmt.Println(function)
	schedulable, exists := p.scheduler.getDeployedFunction(function.ID)
	if exists {
		schedulable.runTime = time.Now()
		return schedulable.worker.SendRequest(request)
	}

	p.scheduler.mux.Lock()
	if len(p.scheduler.undeployed) > 0 {
		name, schedulable := p.scheduler.runUndeployed()
		p.scheduler.mux.Unlock()
		fmt.Printf("chose %s to run func", name)
		schedulable.worker.SendFunction(function.Executable.Code)
		fmt.Println("function sent")
		schedulable.runTime = time.Now()
		result, err := schedulable.worker.SendRequest(request)
		fmt.Println("result obtained")
		p.scheduler.runComplete(name)
		fmt.Println("run complete")
		p.scheduler.scheduleDecommission(name, schedulable)
		fmt.Println("scheduled decomission")
		// Schedule decomission
		return result, err
	} else {
		p.scheduler.mux.Unlock()
		return "", fmt.Errorf("no containers available")
	}
}

func (s *Scheduler) getDeployedFunction(f string) (*ScheduleWorker, bool) {
	s.mux.Lock()
	defer s.mux.Unlock()
	for _, d := range s.deployed {
		if s.workers[d].function == f {
			return s.workers[d], true
		}
	}
	return nil, false
}

func (s *Scheduler) runUndeployed() (string, *ScheduleWorker) {
	var next string
	next, s.undeployed = s.undeployed[0], s.undeployed[1:]
	s.running = append(s.running, next)
	return next, s.workers[next]
}

func (s *Scheduler) runComplete(name string) {
	s.mux.Lock()
	defer s.mux.Unlock()
	var nameIndex int
	for i, w := range s.running {
		if w == name {
			nameIndex = i
			break
		}
	}
	s.running = append(s.running[:nameIndex], s.running[nameIndex+1:]...)
	s.deployed = append(s.deployed, name)
}

func (s *Scheduler) scheduleDecommission(name string, schedulable *ScheduleWorker) {
	go func() {
		for {
			time.Sleep(decomissionTime)
			if time.Since(schedulable.runTime) >= decomissionTime {
				s.mux.Lock()
				nameIndex := -1
				for i, w := range s.deployed {
					if w == name {
						nameIndex = i
						break
					}
				}
				if nameIndex < 0 {
					// Must still be running
					s.mux.Unlock()
					continue
				}

				s.deployed = append(s.deployed[:nameIndex], s.deployed[nameIndex+1:]...)
				s.mux.Unlock()

				err := schedulable.worker.FinishFunction()
				if err != nil {
					fmt.Fprintln(os.Stderr, err.Error())
					return
				}
				err = schedulable.worker.Restore()
				if err != nil {
					fmt.Fprintln(os.Stderr, err.Error())
					return
				}

				s.mux.Lock()
				s.undeployed = append(s.undeployed, name)
				s.mux.Unlock()
				return
			}

		}
	}()
}

package workerpool

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ostenbom/refunction/invoker/types"
	"github.com/ostenbom/refunction/worker"
	log "github.com/sirupsen/logrus"
)

const defaultDecommissionTime = time.Second * 20

type Scheduler struct {
	runtime          string
	workers          map[string]*ScheduleWorker
	undeployed       []string
	deployed         []string
	running          []string
	mux              sync.Mutex
	decommissionTime time.Duration
}

type ScheduleWorker struct {
	worker   *worker.Worker
	runTime  time.Time
	function string
}

func NewScheduler(workers []*worker.Worker, runtime string) *Scheduler {
	scheduleWorkers := make(map[string]*ScheduleWorker)
	var undeployed []string

	for _, w := range workers {
		scheduleWorkers[w.ID] = &ScheduleWorker{
			worker: w,
		}
		undeployed = append(undeployed, w.ID)
	}

	return &Scheduler{
		runtime:          runtime,
		workers:          scheduleWorkers,
		undeployed:       undeployed,
		decommissionTime: defaultDecommissionTime,
	}
}

func NewFakeScheduler(workers map[string]*ScheduleWorker, undeployed []string, decommissionTime time.Duration) *Scheduler {
	return &Scheduler{
		workers:          workers,
		undeployed:       undeployed,
		decommissionTime: decommissionTime,
	}
}

func (s *Scheduler) Run(function *types.FunctionDoc, request interface{}) (interface{}, error) {
	functionLogger := log.WithFields(log.Fields{
		"request":      request,
		"functionID":   function.ID,
		"functionName": function.Name,
		"runtime":      s.runtime,
	})
	name, schedulable, exists := s.RunDeployedFunction(function.ID)
	if exists {
		schedulable.MarkRunTime()

		functionLogger = functionLogger.WithFields(log.Fields{"worker": name})
		functionLogger.Debug("running on deployed worker")
		result, err := schedulable.worker.SendRequest(request)
		functionLogger.WithFields(log.Fields{"result": result}).Debug("response received")

		s.RunComplete(name)
		return result, err
	}

	s.mux.Lock()
	if len(s.undeployed) > 0 {
		name, schedulable := s.RunUndeployed()
		s.mux.Unlock()

		functionCode, err := function.CodeString()
		functionLogger = functionLogger.WithFields(log.Fields{"worker": name, "code": functionCode})
		functionLogger.Debug("loading function")
		if err != nil {
			return "", err
		}
		schedulable.worker.SendFunction(functionCode)
		functionLogger.Debug("sending request")
		// TODO: Set after request response?
		schedulable.MarkRunTime()
		result, err := schedulable.worker.SendRequest(request)
		functionLogger.WithFields(log.Fields{"result": result}).Debug("response received")

		schedulable.SetFunction(function.ID)
		s.RunComplete(name)
		s.ScheduleDecommission(name, schedulable)
		return result, err
	} else {
		s.mux.Unlock()
		s.ForceDecomission()
		return s.Run(function, request)
	}
}

func (s *Scheduler) RunDeployedFunction(f string) (string, *ScheduleWorker, bool) {
	s.mux.Lock()
	defer s.mux.Unlock()
	for i, d := range s.deployed {
		if s.workers[d].GetFunction() == f {
			s.deployed = append(s.deployed[:i], s.deployed[i+1:]...)
			s.running = append(s.running, d)
			return d, s.workers[d], true
		}
	}
	return "", nil, false
}

func (s *Scheduler) RunUndeployed() (string, *ScheduleWorker) {
	var next string
	next, s.undeployed = s.undeployed[0], s.undeployed[1:]
	s.running = append(s.running, next)
	return next, s.workers[next]
}

func (s *Scheduler) RunComplete(name string) {
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

func (s *Scheduler) ScheduleDecommission(name string, schedulable *ScheduleWorker) {
	go func() {
		for {
			time.Sleep(s.decommissionTime)
			if time.Since(schedulable.runTime) >= s.decommissionTime {
				s.mux.Lock()
				nameIndex := -1
				for i, w := range s.deployed {
					if w == name {
						nameIndex = i
						break
					}
				}
				if nameIndex < 0 {
					// Either forced eviction or still running
					nameIndex := -1
					for i, w := range s.undeployed {
						if w == name {
							nameIndex = i
							break
						}
					}
					if nameIndex < 0 {
						// Must still be running
						s.mux.Unlock()
						continue
					} else {
						// Must have been evicted
						s.mux.Unlock()
						return
					}
				}

				s.deployed = append(s.deployed[:nameIndex], s.deployed[nameIndex+1:]...)
				s.mux.Unlock()

				err := schedulable.Decomission()
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

func (s *Scheduler) ForceDecomission() {
	var toDecomission string
	s.mux.Lock()
	if len(s.deployed) <= 0 {
		s.mux.Unlock()
		log.Error("No available decomission slots")
		time.Sleep(time.Millisecond * 100)
		s.ForceDecomission()
		return
	}
	toDecomission, s.deployed = s.deployed[0], s.deployed[1:]
	s.mux.Unlock()

	err := s.workers[toDecomission].Decomission()
	if err != nil {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("Hung container!: could not force decomission: %s", err))
		return
	}

	s.mux.Lock()
	s.undeployed = append(s.undeployed, toDecomission)
	s.mux.Unlock()
}

func (s *Scheduler) End() error {
	var workerErr error
	for _, sw := range s.workers {
		err := sw.worker.End()
		if err != nil {
			workerErr = err
		}
	}
	return workerErr
}

func (sw *ScheduleWorker) Decomission() error {
	// Testing
	if sw.worker == nil {
		return nil
	}

	return sw.worker.Restore()
}

func (sw *ScheduleWorker) MarkRunTime() {
	sw.runTime = time.Now()
}

func (sw *ScheduleWorker) SetFunction(f string) {
	sw.function = f
}

func (sw *ScheduleWorker) GetFunction() string {
	return sw.function
}

// Functions for testing

func (s *Scheduler) DeployedWorkers() []string {
	s.mux.Lock()
	defer s.mux.Unlock()
	d := append([]string{}, s.deployed...)
	return d
}

func (s *Scheduler) UndeployedWorkers() []string {
	s.mux.Lock()
	defer s.mux.Unlock()
	d := append([]string{}, s.undeployed...)
	return d
}

func (s *Scheduler) RunningWorkers() []string {
	s.mux.Lock()
	defer s.mux.Unlock()
	d := append([]string{}, s.running...)
	return d
}

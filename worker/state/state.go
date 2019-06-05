package state

import (
	"fmt"
	"sync"
	"syscall"

	"github.com/ostenbom/refunction/worker/ptrace"
)

type TaskRegState struct {
	task *ptrace.TraceTask
	regs *syscall.PtraceRegs
}

type State struct {
	pid             int
	registers       map[int]TaskRegState
	memoryLocations []*Memory
	fileDescriptors []*FileDescriptor
	rlimits         Rlimits
}

//NewState caller must ensure process stopped before getting state
func NewState(pid int, tasks map[int]*ptrace.TraceTask) (*State, error) {
	var state State
	state.registers = make(map[int]TaskRegState)

	errors := make(chan error)
	results := make(chan TaskRegState, len(tasks))
	var wg sync.WaitGroup
	for _, task := range tasks {
		wg.Add(1)
		task.InStopFunction <- func(t *ptrace.TraceTask) {
			defer wg.Done()
			var regs syscall.PtraceRegs
			err := syscall.PtraceGetRegs(t.Tid, &regs)
			if err != nil {
				errors <- err
			}
			results <- TaskRegState{
				task: t,
				regs: &regs,
			}
		}
	}
	wg.Wait()
	close(results)
	select {
	case err := <-errors:
		return nil, fmt.Errorf("could not get regs: %s", err)
	default:
		break
	}

	for result := range results {
		state.registers[result.task.Tid] = result
	}

	memoryLocations, err := newMemoryLocations(pid)
	if err != nil {
		return nil, fmt.Errorf("could not get memory locations: %s", err)
	}
	state.memoryLocations = memoryLocations

	fileDescriptors, err := newFileDescriptors(pid)
	if err != nil {
		return nil, fmt.Errorf("could not create file descriptor state: %s", err)
	}
	state.fileDescriptors = fileDescriptors

	rlimits, err := newRlimits(pid)
	if err != nil {
		return nil, fmt.Errorf("could not create rlimit state: %s", err)
	}
	state.rlimits = rlimits

	state.pid = pid

	return &state, nil
}

func (s *State) RestoreRegs() error {
	errors := make(chan error)
	var wg sync.WaitGroup
	for tid := range s.registers {
		wg.Add(1)
		regState := s.registers[tid]
		regState.task.InStopFunction <- func(t *ptrace.TraceTask) {
			defer wg.Done()
			err := syscall.PtraceSetRegs(t.Tid, regState.regs)
			if err != nil {
				errors <- err
			}
		}
	}
	wg.Wait()
	select {
	case err := <-errors:
		return fmt.Errorf("could not set regs: %s", err)
	default:
		break
	}
	return nil
}

func (s *State) RestoreProgramBreak() error {
	beforeHeap, err := s.getMemory("[heap]")
	if err != nil {
		return err
	}
	afterOffset, err := s.runSyscall(12, uint64(beforeHeap.endOffset), 0)
	if err != nil {
		return err
	}

	if uint64(beforeHeap.endOffset) != afterOffset {
		return fmt.Errorf("could not return program break to previous value")
	}

	return nil
}

func (s *State) UnmapNewLocations() error {
	currentMemory, err := newMemoryLocations(s.pid)
	if err != nil {
		return fmt.Errorf("could not get new memory on memory changed check: %s", err)
	}

	newLocations := calculateNewLocations(s.memoryLocations, currentMemory)
	for _, loc := range newLocations {
		returnVal, err := s.runSyscall(11, uint64(loc.startOffset), uint64(loc.endOffset-loc.startOffset))
		if err != nil {
			return fmt.Errorf("could not unmap new location: %s", err)
		}
		if returnVal != 0 {
			return fmt.Errorf("could not unmap new location")
		}
	}

	return nil
}

func calculateNewLocations(oldState []*Memory, newState []*Memory) []*Memory {
	var newLocations []*Memory
	for _, newLoc := range newState {
		existed := false
		for _, oldLoc := range oldState {
			if newLoc.name == oldLoc.name && newLoc.startOffset == oldLoc.startOffset {
				existed = true
			}
		}
		if !existed {
			newLocations = append(newLocations, newLoc)
		}
	}

	return newLocations
}

func (s *State) runSyscall(syscallNum uint64, arg1 uint64, arg2 uint64) (uint64, error) {
	// We could work on any thread
	regState := s.chooseAnyRegState()
	regsChan := make(chan syscall.PtraceRegs)
	errorsChan := make(chan error)
	regState.task.InStopFunction <- func(t *ptrace.TraceTask) {
		var regs syscall.PtraceRegs
		err := syscall.PtraceGetRegs(t.Tid, &regs)
		if err != nil {
			errorsChan <- err
		}
		regsChan <- regs
	}

	var currentRegs syscall.PtraceRegs
	select {
	case err := <-errorsChan:
		return 0, fmt.Errorf("could not get regs: %s", err)
	case regs := <-regsChan:
		currentRegs = regs
	}

	// Set registers to correct arguments
	// 12 is brk
	currentRegs.Rax = syscallNum
	currentRegs.Rdi = arg1
	currentRegs.Rsi = arg2
	// currentRegs.Rdx = arg3
	// currentRegs.Rcx = arg4
	// currentRegs.R8 = arg5
	// currentRegs.R9 = arg6

	regState.task.RunSyscall <- currentRegs
	select {
	case returnRegs := <-regState.task.SyscallReturn:
		return returnRegs.Rax, nil
	case err := <-regState.task.SyscallError:
		return 0, err
	}
}

func (s *State) chooseAnyRegState() TaskRegState {
	for tid := range s.registers {
		return s.registers[tid]
	}
	return TaskRegState{}
}

func (s *State) PC() uint64 {
	return s.registers[s.pid].regs.PC()
}

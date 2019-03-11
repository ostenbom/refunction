package state

import (
	"fmt"
	"syscall"
)

type State struct {
	pid             int
	registers       syscall.PtraceRegs
	memoryLocations []*Memory
	fileDescriptors []*FileDescriptor
	rlimits         Rlimits
	stoppedFunction chan func()
}

//NewState caller must ensure process stopped before getting state
func NewState(pid int, stoppedFunction chan func()) (*State, error) {
	var state State

	state.stoppedFunction = stoppedFunction
	done := make(chan error)
	stoppedFunction <- func() {
		err := syscall.PtraceGetRegs(pid, &state.registers)
		done <- err
	}
	err := <-done
	if err != nil {
		return nil, fmt.Errorf("could not get regs: %s", err)
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
	done := make(chan error)
	s.stoppedFunction <- func() {
		err := syscall.PtraceSetRegs(s.pid, &s.registers)
		done <- err
	}
	err := <-done
	if err != nil {
		return fmt.Errorf("could not set regs: %s", err)
	}
	return nil
}

func (s *State) PC() uint64 {
	return s.registers.PC()
}

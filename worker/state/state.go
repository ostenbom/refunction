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
}

//NewState caller must ensure process stopped before getting state
func NewState(pid int) (*State, error) {
	var state State
	err := syscall.PtraceGetRegs(pid, &state.registers)
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

	state.pid = pid

	return &state, nil
}

func (s *State) RestoreRegs() error {
	err := syscall.PtraceSetRegs(s.pid, &s.registers)
	if err != nil {
		return fmt.Errorf("could not set regs: %s", err)
	}
	return nil
}

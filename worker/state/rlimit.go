package state

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/unix"
)

type Rlimits map[int]*unix.Rlimit

func newRlimits(pid int) (Rlimits, error) {
	rlimits := make(Rlimits)
	rlimit := new(unix.Rlimit)

	err := prlimit(pid, unix.RLIMIT_AS, rlimit, nil)
	if err != nil {
		return nil, fmt.Errorf("could not get rlimit: %s", err)
	}
	rlimits[unix.RLIMIT_AS] = rlimit

	return rlimits, nil
}

func prlimit(pid int, resource int, oldRlimit *unix.Rlimit, newRlimit *unix.Rlimit) error {
	_, _, errNo := unix.RawSyscall6(unix.SYS_PRLIMIT64, uintptr(pid), uintptr(resource), uintptr(unsafe.Pointer(newRlimit)), uintptr(unsafe.Pointer(oldRlimit)), 0, 0)
	if errNo != 0 {
		return errNo
	}

	return nil
}

func (s *State) GetRlimits() Rlimits {
	return s.rlimits
}

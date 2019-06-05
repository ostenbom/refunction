package ptrace

import (
	"fmt"
	"runtime"
	"strings"
	"syscall"

	"github.com/ostenbom/refunction/worker/safewriter"
	sec "github.com/seccomp/libseccomp-golang"
	log "github.com/sirupsen/logrus"
)

type TraceTask struct {
	Tid            int
	Gid            int
	SignalStop     chan syscall.WaitStatus
	Continue       chan syscall.Signal
	HasContinued   chan int
	Detach         chan int
	HasDetached    chan int
	RunSyscall     chan syscall.PtraceRegs
	SyscallReturn  chan syscall.PtraceRegs
	SyscallError   chan error
	InStopFunction chan func(*TraceTask)
	Error          chan error
	attachOptions  []int
	straceEnabled  bool
	writer         *safewriter.SafeWriter
}

func NewTraceTask(tid int, gid int, attachOptions []int, straceEnabled bool, writer *safewriter.SafeWriter) (*TraceTask, error) {
	task := &TraceTask{
		Tid:            tid,
		Gid:            gid,
		SignalStop:     make(chan syscall.WaitStatus, 1),
		Continue:       make(chan syscall.Signal),
		HasContinued:   make(chan int),
		Detach:         make(chan int),
		HasDetached:    make(chan int),
		RunSyscall:     make(chan syscall.PtraceRegs),
		SyscallReturn:  make(chan syscall.PtraceRegs),
		SyscallError:   make(chan error),
		InStopFunction: make(chan func(*TraceTask)),
		Error:          make(chan error),
		attachOptions:  attachOptions,
		straceEnabled:  straceEnabled,
		writer:         writer,
	}

	err := task.ptraceLoop()
	if err != nil {
		return nil, err
	}

	task.awaitPtraceError()
	return task, nil
}

func (t *TraceTask) ptraceLoop() error {
	attachErr := make(chan error)
	go func() {
		// Crucial: trying to call ptrace functions from a different thread
		// than the attacher causes undefined behaviour.
		// LockOSTread only locks current Goroutine. All ptrace functions
		// must therefore be called from here.
		runtime.LockOSThread()

		err := t.ptraceAttach()

		// After this point errors are handled by a separate Goroutine
		attachErr <- err
		if err != nil {
			runtime.UnlockOSThread()
			return
		}

		enteringSyscall := true

		var waitStat syscall.WaitStatus
		for {
			_, err := syscall.Wait4(t.Tid, &waitStat, syscall.WALL, nil)
			if err != nil {
				if waitStat.Exited() {
					break
				}

				t.Error <- fmt.Errorf("error waiting for child: %s", err)
				break
			}
			log.Debug("child waited for")

			if waitStat>>16 == PTRACE_EVENT_STOP {
				fmt.Printf("Observed group stop: %d\n", t.Tid)
				t.SignalStop <- PTRACE_EVENT_STOP
				continuePtrace, err := t.awaitContinueOrders()
				log.Debug("completed wait")
				if !continuePtrace {
					if err != nil {
						t.Error <- err
					}
					break
				}
				continue
			}

			if !waitStat.Stopped() {
				// TODO: Continue here?
				fmt.Println("did not wait for a stopped signal!")
				break
			}

			if waitStat.StopSignal() == syscall.SIGTRAP|0x80 {
				if enteringSyscall {
					err = t.printSyscall()
					if err != nil {
						t.Error <- fmt.Errorf("could not print syscall: %s", err)
						break
					}
				}

				err = t.continueTrace(0)
				if err != nil {
					t.Error <- fmt.Errorf("could not continue after syscall stop: %s", err)
					break
				}

				enteringSyscall = !enteringSyscall
			} else {
				t.SignalStop <- waitStat
				log.WithFields(log.Fields{
					"StopSignal": waitStat.StopSignal(),
					"ExitSignal": waitStat.ExitStatus(),
					"Signal":     waitStat.Signal(),
				}).Debug("awaiting continue orders")
				continuePtrace, err := t.awaitContinueOrders()
				log.Debug("completed wait")
				if !continuePtrace {
					if err != nil {
						t.Error <- err
					}
					break
				}
			}
		}

		runtime.UnlockOSThread()
	}()

	return <-attachErr
}

func (t *TraceTask) ptraceAttach() error {
	var opts int
	for _, opt := range t.attachOptions {
		opts = opts | opt
	}

	err := PtraceSeize(t.Tid, opts)
	if err != nil {
		return fmt.Errorf("could not attach: %s", err)
	}

	return nil
}

func (t *TraceTask) awaitContinueOrders() (bool, error) {
	for {
		select {
		case continueSignal := <-t.Continue:
			t.popWait()
			err := t.continueTrace(continueSignal)
			if err != nil {
				return false, fmt.Errorf("could not continue after syscall stop: %s", err)
			}
			t.HasContinued <- 1
			return true, nil
		case regs := <-t.RunSyscall:
			returnRegs, err := t.runSyscall(regs)
			if err != nil {
				t.SyscallError <- err
			} else {
				t.SyscallReturn <- returnRegs
			}
		case <-t.Detach:
			err := syscall.PtraceDetach(t.Tid)
			if err != nil {
				return false, fmt.Errorf("could not detach: %s", err)
			}
			t.HasDetached <- 1
			return false, nil
		case f := <-t.InStopFunction:
			f(t)
		}
	}
}

func (t *TraceTask) popWait() {
	select {
	case <-t.SignalStop:
		break
	default:
		break
	}
}

func (t *TraceTask) runSyscall(argRegs syscall.PtraceRegs) (syscall.PtraceRegs, error) {
	// Get current registers
	var startRegs syscall.PtraceRegs
	err := syscall.PtraceGetRegs(t.Tid, &startRegs)
	if err != nil {
		return syscall.PtraceRegs{}, fmt.Errorf("could not get task regs: %s", err)
	}

	// Get current Rip data
	preSyscallInstruction := make([]byte, 2)
	count, err := syscall.PtracePeekData(t.Tid, uintptr(startRegs.PC()), preSyscallInstruction)
	if err != nil || count != 2 {
		return syscall.PtraceRegs{}, fmt.Errorf("could not peek data: %s", err)
	}

	// Change instruction at Rip to be 0f 05 (syscall)
	syscallInstruction := []byte{byte(0x0f), byte(0x05)}
	count, err = syscall.PtracePokeData(t.Tid, uintptr(startRegs.PC()), syscallInstruction)
	if err != nil || count != 2 {
		return syscall.PtraceRegs{}, fmt.Errorf("could not poke instruction data: %s", err)
	}

	syscallRegs := startRegs

	syscallRegs.Rax = argRegs.Rax
	syscallRegs.Rdi = argRegs.Rdi
	syscallRegs.Rsi = argRegs.Rsi
	// syscallRegs.Rdx = argRegs.Rdx
	// syscallRegs.Rcx = argRegs.Rcx
	// syscallRegs.R8 = argRegs.R8
	// syscallRegs.R9 = argRegs.R9

	// Change regs for syscall
	err = syscall.PtraceSetRegs(t.Tid, &syscallRegs)
	if err != nil {
		return syscall.PtraceRegs{}, fmt.Errorf("could not poke instruction data: %s", err)
	}

	// Continue to stop at next syscall
	err = syscall.PtraceSingleStep(t.Tid)
	if err != nil {
		return syscall.PtraceRegs{}, fmt.Errorf("could not continue task: %s", err)
	}

	// Let enter syscall
	var waitStat syscall.WaitStatus
	// Catch exit. Change Rip back. Set instruction at Rip back to what it was.
	_, err = syscall.Wait4(t.Tid, &waitStat, syscall.WALL, nil)
	if err != nil {
		return syscall.PtraceRegs{}, fmt.Errorf("could wait on syscall task: %s", err)
	}

	var exitRegs syscall.PtraceRegs
	err = syscall.PtraceGetRegs(t.Tid, &exitRegs)
	if err != nil {
		return syscall.PtraceRegs{}, fmt.Errorf("could not get task regs: %s", err)
	}

	count, err = syscall.PtracePokeData(t.Tid, uintptr(startRegs.PC()), preSyscallInstruction)
	if err != nil || count != 2 {
		return syscall.PtraceRegs{}, fmt.Errorf("could not poke instruction data: %s", err)
	}

	err = syscall.PtraceSetRegs(t.Tid, &startRegs)
	if err != nil {
		return syscall.PtraceRegs{}, fmt.Errorf("could not reset task regs: %s", err)
	}

	return exitRegs, nil
}

func (t *TraceTask) continueTrace(signal syscall.Signal) error {
	var err error
	if t.straceEnabled {
		err = syscall.PtraceSyscall(t.Tid, int(signal))
	} else {
		err = syscall.PtraceCont(t.Tid, int(signal))
	}

	if err != nil {
		return err
	}

	return nil
}

func (t *TraceTask) awaitPtraceError() {
	go func() {
		err := <-t.Error
		fmt.Println(err)
	}()
}

func (t *TraceTask) printSyscall() error {
	var regs syscall.PtraceRegs
	err := syscall.PtraceGetRegs(t.Tid, &regs)
	if err != nil {
		return fmt.Errorf("cound not get regs: %s", err)
	}

	name, err := sec.ScmpSyscall(regs.Orig_rax).GetName()
	if err == nil {
		err = t.straceWrite(fmt.Sprintf("syscall: %s\n", name))
		if err != nil {
			return fmt.Errorf("strace write err: %s", err)
		}
	} else {
		err = t.straceWrite(fmt.Sprintf("unknown syscall: %d\n", regs.Orig_rax))
		if err != nil {
			return fmt.Errorf("strace write err: %s", err)
		}
	}

	return nil
}

func (t *TraceTask) straceWrite(out string) error {
	b := strings.NewReader(out)

	err := t.writer.Write(b)
	if err != nil {
		return fmt.Errorf("could not print to strace output: %s", err)
	}

	return nil
}

func (t *TraceTask) Stop() error {
	select {
	case signal := <-t.SignalStop:
		// If it's already stopped for some reason that's fine
		t.SignalStop <- signal
		return nil
	default:
		break
	}

	err := syscall.Tgkill(t.Gid, t.Tid, syscall.SIGSTOP)
	if err != nil {
		return err
	}

	stop := <-t.SignalStop
	t.SignalStop <- stop
	return nil
}

// PTRACE_SEIZE from linux kernel https://github.com/torvalds/linux/blob/d8a5b80568a9cb66810e75b182018e9edb68e8ff/include/uapi/linux/ptrace.h#L53
const (
	PTRACE_SEIZE      = 0x4206
	PTRACE_EVENT_STOP = 128
)

func PtraceSeize(pid int, opts int) (err error) {
	return ptrace(PTRACE_SEIZE, pid, 0, uintptr(opts))
}

func ptrace(request int, pid int, addr uintptr, data uintptr) (err error) {
	_, _, e1 := syscall.Syscall6(syscall.SYS_PTRACE, uintptr(request), uintptr(pid), uintptr(addr), uintptr(data), 0, 0)
	if e1 != 0 {
		err = e1
	}
	return
}

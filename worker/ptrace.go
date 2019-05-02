package worker

import (
	"fmt"
	"io"
	"runtime"
	"strings"
	"syscall"

	sec "github.com/seccomp/libseccomp-golang"
	log "github.com/sirupsen/logrus"
)

func (m *Worker) Attach() error {
	err := m.ptraceLoop()
	if err != nil {
		return err
	}

	m.awaitPtraceError()
	return nil
}

func (m *Worker) ptraceLoop() error {
	attachErr := make(chan error)
	go func() {
		// Crucial: trying to call ptrace functions from a different thread
		// than the attacher causes undefined behaviour.
		// LockOSTread only locks current Goroutine. All ptrace functions
		// must therefore be called from here.
		runtime.LockOSThread()

		err := m.ptraceAttach()

		// After this point errors are handled by a separate Goroutine
		attachErr <- err
		if err != nil {
			runtime.UnlockOSThread()
			return
		}

		continuePtrace, err := m.awaitContinueOrders()
		if !continuePtrace {
			if err != nil {
				m.ptrace.Error <- fmt.Errorf("could not continue after attach: %s", err)
			}
			runtime.UnlockOSThread()
			return
		}

		enteringSyscall := true

		var waitStat syscall.WaitStatus
		for {
			_, err := syscall.Wait4(int(m.task.Pid()), &waitStat, 0, nil)
			if err != nil {
				if waitStat.Exited() {
					break
				}

				m.ptrace.Error <- fmt.Errorf("error waiting for child: %s", err)
				break
			}
			log.Debug("child waited for")

			if !waitStat.Stopped() {
				// TODO: Continue here?
				fmt.Println("did not wait for a stopped signal!")
				break
			}

			if waitStat.StopSignal() == syscall.SIGTRAP|0x80 {
				if enteringSyscall {
					err = m.printSyscall()
					if err != nil {
						m.ptrace.Error <- fmt.Errorf("could not print syscall: %s", err)
						break
					}
				}

				err = m.ptraceContinue(0)
				if err != nil {
					m.ptrace.Error <- fmt.Errorf("could not continue after syscall stop: %s", err)
					break
				}

				enteringSyscall = !enteringSyscall
			} else {
				m.ptrace.SignalStop <- waitStat
				log.WithFields(log.Fields{
					"StopSignal": waitStat.StopSignal(),
					"ExitSignal": waitStat.ExitStatus(),
					"Signal":     waitStat.Signal(),
				}).Debug("awaiting continue orders")
				continuePtrace, err := m.awaitContinueOrders()
				log.Debug("completed wait")
				if !continuePtrace {
					if err != nil {
						m.ptrace.Error <- err
					}
					break
				}
			}
		}

		runtime.UnlockOSThread()
	}()

	return <-attachErr
}

func (m *Worker) ptraceAttach() error {
	err := syscall.PtraceAttach(int(m.task.Pid()))
	if err != nil {
		return fmt.Errorf("could not attach: %s", err)
	}

	m.attached = true

	var waitStat syscall.WaitStatus
	_, err = syscall.Wait4(int(m.task.Pid()), &waitStat, 0, nil)
	if err != nil {
		return fmt.Errorf("could not wait for attach to complete: %s", err)
	}
	m.ptrace.SignalStop <- waitStat

	var opts int
	for _, opt := range m.attachOptions {
		opts = opts | opt
	}

	err = syscall.PtraceSetOptions(int(m.task.Pid()), opts)
	if err != nil {
		return fmt.Errorf("could not set ptrace options: %s", err)
	}

	return nil
}

func (m *Worker) awaitContinueOrders() (bool, error) {
	for {
		select {
		case continueSignal := <-m.ptrace.Continue:
			m.ptracePopWait()
			err := m.ptraceContinue(continueSignal)
			if err != nil {
				return false, fmt.Errorf("could not continue after syscall stop: %s", err)
			}
			m.ptrace.HasContinued <- 1
			return true, nil
		case <-m.ptrace.Detach:
			err := syscall.PtraceDetach(int(m.task.Pid()))
			if err != nil {
				return false, fmt.Errorf("could not detach: %s", err)
			}
			m.ptrace.HasDetached <- 1
			return false, nil
		case f := <-m.ptrace.InStopFunction:
			f()
		}
	}
}

func (m *Worker) ptracePopWait() {
	select {
	case <-m.ptrace.SignalStop:
		break
	default:
		break
	}
}

func (m *Worker) ptraceContinue(signal syscall.Signal) error {
	var err error
	if m.straceEnabled {
		err = syscall.PtraceSyscall(int(m.task.Pid()), int(signal))
	} else {
		err = syscall.PtraceCont(int(m.task.Pid()), int(signal))
	}

	if err != nil {
		return err
	}

	return nil
}

func (m *Worker) awaitPtraceError() {
	go func() {
		err := <-m.ptrace.Error
		fmt.Println(err)
		m.End()
	}()
}

func (m *Worker) printSyscall() error {
	var regs syscall.PtraceRegs
	err := syscall.PtraceGetRegs(int(m.task.Pid()), &regs)
	if err != nil {
		return fmt.Errorf("cound not get regs: %s", err)
	}

	name, err := sec.ScmpSyscall(regs.Orig_rax).GetName()
	if err == nil {
		err = m.straceWrite(fmt.Sprintf("syscall: %s\n", name))
		if err != nil {
			return fmt.Errorf("strace write err: %s", err)
		}
	} else {
		err = m.straceWrite(fmt.Sprintf("unknown syscall: %d\n", regs.Orig_rax))
		if err != nil {
			return fmt.Errorf("strace write err: %s", err)
		}
	}

	return nil
}

func (m *Worker) straceWrite(out string) error {
	b := strings.NewReader(out)

	_, err := io.Copy(m.straceOutput, b)
	if err != nil {
		return fmt.Errorf("could not print to strace output: %s", err)
	}

	return nil
}

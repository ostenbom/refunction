package worker

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"runtime"
	"syscall"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
)

func NewWorker(id string, client *containerd.Client, childID string, image string) (*Worker, error) {
	ctx := namespaces.WithNamespace(context.Background(), "refunction-worker"+id)

	return &Worker{
		ID:      id,
		childID: childID,
		image:   image,
		client:  client,
		ctx:     ctx,
	}, nil
}

type Worker struct {
	ID           string
	childID      string
	image        string
	client       *containerd.Client
	ctx          context.Context
	container    containerd.Container
	task         containerd.Task
	taskExitChan <-chan containerd.ExitStatus
	attached     bool
	stopped      bool
}

func (m *Worker) StartChild() error {
	image, err := m.client.Pull(m.ctx, m.image, containerd.WithPullUnpack)
	if err != nil {
		return err
	}

	containerID := m.childID + "-" + m.ID
	container, err := m.client.NewContainer(
		m.ctx,
		containerID,
		containerd.WithNewSnapshot(containerID, image),
		containerd.WithNewSpec(oci.WithImageConfig(image)),
	)
	if err != nil {
		return err
	}

	m.container = container

	task, err := container.NewTask(m.ctx, cio.NullIO)
	if err != nil {
		return err
	}
	m.task = task

	taskExitChan, err := task.Wait(m.ctx)
	if err != nil {
		return err
	}
	m.taskExitChan = taskExitChan

	if err := task.Start(m.ctx); err != nil {
		return err
	}

	m.attached = false
	m.stopped = false

	return nil
}

func (m *Worker) Attach() error {
	// Crucial: trying to detach from a different thread
	// than the attacher causes undefined behaviour
	runtime.LockOSThread()
	err := syscall.PtraceAttach(int(m.task.Pid()))
	if err != nil {
		return err
	}

	m.attached = true
	m.stopped = true

	_, err = syscall.Wait4(int(m.task.Pid()), nil, 0, nil)
	return err
}

func (m *Worker) Detach() error {
	if !m.stopped {
		err := m.Stop()
		if err != nil {
			return fmt.Errorf("could not stop child for detach: %s", err)
		}
	}

	err := syscall.PtraceDetach(int(m.task.Pid()))
	if err != nil {
		return err
	}

	m.attached = false
	runtime.UnlockOSThread()
	return nil
}

func (m *Worker) Stop() error {
	err := syscall.Kill(int(m.task.Pid()), syscall.SIGSTOP)
	if err != nil {
		return err
	}

	_, err = syscall.Wait4(int(m.task.Pid()), nil, 0, nil)
	m.stopped = true
	return err
}

func (m *Worker) Continue() error {
	m.stopped = false
	return syscall.PtraceCont(int(m.task.Pid()), 0)
}

func (m *Worker) SendEnableSignal() error {
	pid, err := m.ChildPid()
	if err != nil {
		return err
	}

	err = syscall.Kill(int(pid), syscall.SIGUSR1)
	if err != nil {
		return err
	}

	// If not attached, signal will go through
	if !m.attached {
		return nil
	}

	var waitStat syscall.WaitStatus
	_, err = syscall.Wait4(int(pid), &waitStat, 0, nil)

	if err != nil {
		return err
	}
	if !waitStat.Stopped() {
		return errors.New("child not stopped after signal")
	}

	return syscall.PtraceCont(int(pid), int(waitStat.StopSignal()))
}

func (m *Worker) GetState() (*State, error) {
	if !m.stopped {
		err := m.Stop()
		if err != nil {
			return nil, fmt.Errorf("could not stop child for regs print: %s", err)
		}
		defer m.Continue()
	}

	var state State
	err := syscall.PtraceGetRegs(int(m.task.Pid()), &state.registers)
	if err != nil {
		return nil, fmt.Errorf("could not get regs: %s", err)
	}

	memoryLocations, err := NewMemoryLocations(int(m.task.Pid()))
	if err != nil {
		return nil, fmt.Errorf("could not get memory locations: %s", err)
	}
	state.memoryLocations = memoryLocations

	return &state, nil
}

func (m *Worker) GetImage(name string) (containerd.Image, error) {
	return m.client.GetImage(m.ctx, name)
}

func (m *Worker) ListImages() ([]containerd.Image, error) {
	return m.client.ListImages(m.ctx)
}

func (m *Worker) ChildPid() (uint32, error) {
	if m.task == nil {
		return 0, errors.New("child not initialized")
	}

	return m.task.Pid(), nil
}

func (m *Worker) PrintStack() error {
	return m.printProcFile("stack")
}

func (m *Worker) PrintMaps() error {
	return m.printProcFile("maps")
}

func (m *Worker) printProcFile(fileName string) error {
	stack, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/%s", int(m.task.Pid()), fileName))
	if err != nil {
		return fmt.Errorf("could not read child /proc/pid/%s file: %s", fileName, err)
	}

	fmt.Println(string(stack))
	return nil
}

func (m *Worker) PrintRegs() error {
	err := m.Stop()
	if err != nil {
		return fmt.Errorf("could not stop child for regs print: %s", err)
	}
	defer m.Continue()

	var regs syscall.PtraceRegs
	err = syscall.PtraceGetRegs(int(m.task.Pid()), &regs)
	if err != nil {
		return fmt.Errorf("could not get regs: %s", err)
	}

	fmt.Printf("Regs: %+v\n", regs)

	return nil
}

func (m *Worker) End() error {
	if m.task != nil {
		if err := m.task.Kill(m.ctx, syscall.SIGKILL); err != nil {
			if errdefs.IsFailedPrecondition(err) || errdefs.IsNotFound(err) {
				return nil
			}
			return fmt.Errorf("failed to kill in manager end: %s", err)
		}

		<-m.taskExitChan

		_, err := m.task.Delete(m.ctx)
		if err != nil {
			return err
		}
	}

	if m.container != nil {
		m.container.Delete(m.ctx, containerd.WithSnapshotCleanup)
	}

	return nil
}

func (m *Worker) CleanSnapshot(name string) error {
	sservice := m.client.SnapshotService("overlayfs")
	return sservice.Remove(m.ctx, name)
}

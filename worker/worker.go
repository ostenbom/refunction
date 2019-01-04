package worker

import (
	"context"
	"errors"
	"syscall"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
)

func NewManager(id string, client *containerd.Client, childID string, image string) (*Manager, error) {
	ctx := namespaces.WithNamespace(context.Background(), "refunction-worker"+id)

	return &Manager{
		ID:      id,
		childID: childID,
		image:   image,
		client:  client,
		ctx:     ctx,
	}, nil
}

type Manager struct {
	ID           string
	childID      string
	image        string
	client       *containerd.Client
	ctx          context.Context
	container    containerd.Container
	task         containerd.Task
	taskExitChan <-chan containerd.ExitStatus
	attached     bool
}

func (m *Manager) StartChild() error {
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

	return nil
}

func (m *Manager) AttachChild() error {
	err := syscall.PtraceAttach(int(m.task.Pid()))
	if err != nil {
		return err
	}

	m.attached = true

	_, err = syscall.Wait4(int(m.task.Pid()), nil, 0, nil)
	return err
}

func (m *Manager) DetachChild() error {
	err := syscall.PtraceDetach(int(m.task.Pid()))
	if err != nil {
		return err
	}

	m.attached = false
	return nil
}

func (m *Manager) EnterTraceStop() error {
	err := syscall.Kill(int(m.task.Pid()), syscall.SIGSTOP)
	if err != nil {
		return err
	}

	_, err = syscall.Wait4(int(m.task.Pid()), nil, 0, nil)
	return err
}

func (m *Manager) SendEnableSignal() error {
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

func (m *Manager) ContinueChild() error {
	return syscall.PtraceCont(int(m.task.Pid()), 0)
}

func (m *Manager) GetImage(name string) (containerd.Image, error) {
	return m.client.GetImage(m.ctx, name)
}

func (m *Manager) ListImages() ([]containerd.Image, error) {
	return m.client.ListImages(m.ctx)
}

func (m *Manager) ChildPid() (uint32, error) {
	if m.task == nil {
		return 0, errors.New("child not initialized")
	}

	return m.task.Pid(), nil
}

func (m *Manager) End() error {
	if m.task != nil {
		err := m.task.Kill(m.ctx, syscall.SIGKILL)
		if err != nil {
			return err
		}

		<-m.taskExitChan

		_, err = m.task.Delete(m.ctx)
		if err != nil {
			return err
		}
	}

	if m.container != nil {
		m.container.Delete(m.ctx, containerd.WithSnapshotCleanup)
	}

	return nil
}

func (m *Manager) CleanSnapshot(name string) error {
	sservice := m.client.SnapshotService("overlayfs")
	return sservice.Remove(m.ctx, name)
}

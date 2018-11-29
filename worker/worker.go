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

func NewManager(id string, client *containerd.Client) (*Manager, error) {
	ctx := namespaces.WithNamespace(context.Background(), "refunction-worker"+id)

	return &Manager{
		Id:     id,
		client: client,
		ctx:    ctx,
	}, nil
}

type Manager struct {
	Id           string
	client       *containerd.Client
	ctx          context.Context
	container    containerd.Container
	task         containerd.Task
	taskExitChan <-chan containerd.ExitStatus
}

func (m *Manager) StartChild() error {
	image, err := m.client.Pull(m.ctx, "docker.io/ostenbom/ptrace-sleep:latest", containerd.WithPullUnpack)
	if err != nil {
		return err
	}

	containerId := "ptrace-sleep-" + m.Id
	container, err := m.client.NewContainer(
		m.ctx,
		containerId,
		containerd.WithNewSnapshotView(containerId, image),
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

	return nil
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
		err := m.task.Kill(m.ctx, syscall.SIGTERM)
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

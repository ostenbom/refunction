package worker

import (
	"context"
	"syscall"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
)

func NewManager() (*Manager, error) {
	client, err := ContainerdClient()
	if err != nil {
		return nil, err
	}

	ctx := namespaces.WithNamespace(context.Background(), "worker-test")

	return &Manager{
		client: client,
		ctx:    ctx,
	}, nil
}

type Manager struct {
	client       *containerd.Client
	ctx          context.Context
	container    containerd.Container
	task         containerd.Task
	taskExitChan <-chan containerd.ExitStatus
}

func (m *Manager) StartChild() error {
	image, err := m.client.Pull(m.ctx, "docker.io/library/redis:alpine", containerd.WithPullUnpack)
	if err != nil {
		return err
	}

	// create a container
	container, err := m.client.NewContainer(
		m.ctx,
		"redis-server",
		containerd.WithImage(image),
		containerd.WithNewSpec(oci.WithImageConfig(image)),
	)
	if err != nil {
		return err
	}

	m.container = container

	task, err := container.NewTask(m.ctx, cio.NewCreator(cio.WithStdio))
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

func (m *Manager) ListImages() ([]containerd.Image, error) {
	return m.client.ListImages(m.ctx)
}

func (m *Manager) End() error {
	if m.task != nil {
		err := m.task.Kill(m.ctx, syscall.SIGTERM)
		if err != nil {
			return err
		}

		status := <-m.taskExitChan
		_, _, err = status.Result()
		if err != nil {
			return err
		}
	}

	if m.container != nil {
		m.container.Delete(m.ctx)
	}

	m.client.Close()
	return nil
}

func ContainerdClient() (*containerd.Client, error) {
	return containerd.New("/run/containerd/containerd.sock")
}

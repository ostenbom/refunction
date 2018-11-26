package worker

import (
	"time"

	"github.com/containerd/containerd"
)

func NewWorker() *Worker {
	return &Worker{
		StartedAt: time.Now(),
	}
}

type Worker struct {
	StartedAt time.Time
}

func ContainerdClient() (*containerd.Client, error) {
	return containerd.New("/run/containerd/containerd.sock")
}

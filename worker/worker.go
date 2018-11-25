package worker

import "time"

func NewWorker() *Worker {
	return &Worker{
		StartedAt: time.Now(),
	}
}

type Worker struct {
	StartedAt time.Time
}

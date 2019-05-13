package worker

import (
	"context"
	"syscall"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/api/types"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/ostenbom/refunction/worker/ptrace"
)

func NewLocalWorker(id string, pid uint32) *Worker {
	ctx := namespaces.WithNamespace(context.Background(), "refunction-worker"+id)
	return &Worker{
		ID:             id,
		targetSnapshot: "",
		runtime:        "",
		ctx:            ctx,
		creator:        cio.NullIO,
		task: &localTask{
			id:  id,
			pid: pid,
		},
		traceTasks:    make(map[int]*ptrace.TraceTask),
		straceEnabled: false,
	}
}

type localTask struct {
	id  string
	pid uint32
}

func (lt *localTask) ID() string {
	return lt.id
}

// Pid is the system specific process id
func (lt *localTask) Pid() uint32 {
	return lt.pid
}

// Start starts the process executing the user's defined binary
func (lt *localTask) Start(context.Context) error {
	return nil
}

// Delete removes the process and any resources allocated returning the exit status
func (lt *localTask) Delete(context.Context, ...containerd.ProcessDeleteOpts) (*containerd.ExitStatus, error) {
	return nil, nil
}

// Kill sends the provided signal to the process
func (lt *localTask) Kill(context.Context, syscall.Signal, ...containerd.KillOpts) error {
	return nil
}

// Wait asynchronously waits for the process to exit, and sends the exit code to the returned channel
func (lt *localTask) Wait(context.Context) (<-chan containerd.ExitStatus, error) {
	c := make(chan containerd.ExitStatus)
	return c, nil
}

// CloseIO allows various pipes to be closed on the process
func (lt *localTask) CloseIO(context.Context, ...containerd.IOCloserOpts) error {
	return nil
}

// Resize changes the width and heigh of the process's terminal
func (lt *localTask) Resize(ctx context.Context, w, h uint32) error {
	return nil
}

// IO returns the io set for the process
func (lt *localTask) IO() cio.IO {
	return nil
}

// Status returns the executing status of the process
func (lt *localTask) Status(context.Context) (containerd.Status, error) {
	return containerd.Status{
		Status: "running",
	}, nil
}

// Pause suspends the execution of the task
func (lt *localTask) Pause(context.Context) error {
	return nil
}

// Resume the execution of the task
func (lt *localTask) Resume(context.Context) error {
	return nil
}

// Exec creates a new process inside the task
func (lt *localTask) Exec(context.Context, string, *specs.Process, cio.Creator) (containerd.Process, error) {
	return lt, nil
}

// Pids returns a list of system specific process ids inside the task
func (lt *localTask) Pids(context.Context) ([]containerd.ProcessInfo, error) {
	return []containerd.ProcessInfo{}, nil
}

// Checkpoint serializes the runtime and memory information of a task into an
// OCI Index that can be push and pulled from a remote resource.
//
// Additional software like CRIU maybe required to checkpoint and restore tasks
func (lt *localTask) Checkpoint(context.Context, ...containerd.CheckpointTaskOpts) (containerd.Image, error) {
	return nil, nil
}

// Update modifies executing tasks with updated settings
func (lt *localTask) Update(context.Context, ...containerd.UpdateTaskOpts) error {
	return nil
}

// LoadProcess loads a previously created exec'd process
func (lt *localTask) LoadProcess(context.Context, string, cio.Attach) (containerd.Process, error) {
	return lt, nil
}

// Metrics returns task metrics for runtime specific metrics
//
// The metric types are generic to containerd and change depending on the runtime
// For the built in Linux runtime, github.com/containerd/cgroups.Metrics
// are returned in protobuf format
func (lt *localTask) Metrics(context.Context) (*types.Metric, error) {
	return nil, nil
}

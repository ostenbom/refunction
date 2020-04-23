package worker

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"syscall"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/ostenbom/refunction/controller"
	. "github.com/ostenbom/refunction/state"
)

func NewWorker(id string, client *containerd.Client, runtime, targetSnapshot string) (*Worker, error) {
	ctx := namespaces.WithNamespace(context.Background(), "refunction-worker"+id)

	snapManager, err := NewSnapshotManager(ctx, client, runtime)
	if err != nil {
		return nil, err
	}

	err = snapManager.CreateLayerFromBase(targetSnapshot)
	if err != nil {
		return nil, err
	}

	return NewWorkerWithSnapManager(id, client, runtime, targetSnapshot, snapManager, ctx)
}

func NewWorkerWithSnapManager(id string, client *containerd.Client, runtime, targetSnapshot string, snapManager *SnapshotManager, ctx context.Context) (*Worker, error) {

	return &Worker{
		ID:             id,
		controller:     controller.NewController(),
		targetSnapshot: targetSnapshot,
		runtime:        runtime,
		client:         client,
		ctx:            ctx,
		creator:        cio.NullIO,
		snapManager:    snapManager,
	}, nil
}

type Worker struct {
	ID             string
	ContainerID    string
	targetSnapshot string
	runtime        string
	controller     controller.Controller
	stderrWriters  []io.Writer
	stdoutWriters  []io.Writer
	client         *containerd.Client
	ctx            context.Context
	creator        cio.Creator
	snapManager    *SnapshotManager
	container      containerd.Container
	task           containerd.Task
	taskExitChan   <-chan containerd.ExitStatus
	IP             net.IP
}

func (m *Worker) WithStdPipes(stderrWriter io.Writer, stdoutWriters ...io.Writer) {
	m.stderrWriters = []io.Writer{stderrWriter}
	m.stdoutWriters = stdoutWriters
}

func (m *Worker) connectStdPipes() {
	stdinRead, stdinWrite := io.Pipe()
	stdoutRead, stdoutWrite := io.Pipe()
	stderrRead, stderrWrite := io.Pipe()

	var collectedStdErr io.Writer
	if len(m.stderrWriters) > 0 {
		collectedStdErr = io.MultiWriter(append(m.stderrWriters, stderrWrite)...)
	} else {
		collectedStdErr = stderrWrite
	}

	var collectedStdOut io.Writer
	if len(m.stdoutWriters) > 0 {
		collectedStdOut = io.MultiWriter(append(m.stdoutWriters, stdoutWrite)...)
	} else {
		collectedStdOut = stdoutWrite
	}

	m.creator = cio.NewCreator(cio.WithStreams(stdinRead, collectedStdOut, collectedStdErr))

	m.controller.SetStreams(stdinWrite, stdoutRead, stderrRead)

	go func() {
		io.Copy(os.Stderr, stderrRead)
	}()
}

func (m *Worker) WithSyscallTrace(to io.Writer) {
	m.controller.WithSyscallTrace(to)
}

func WithNetNsHook(ipFile string) oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *oci.Spec) error {
		s.Hooks = &specs.Hooks{
			Prestart: []specs.Hook{specs.Hook{
				Path: "/usr/local/bin/netns",
				Args: []string{"netns", "--ipfile", ipFile},
			}},
		}
		return nil
	}
}

func WithDefaultMemoryLimit(_ context.Context, _ oci.Client, _ *containers.Container, s *oci.Spec) error {
	if s.Linux == nil {
		s.Linux = &specs.Linux{}
	}
	if s.Linux.Resources == nil {
		s.Linux.Resources = &specs.LinuxResources{}
	}
	if s.Linux.Resources.Memory == nil {
		s.Linux.Resources.Memory = &specs.LinuxMemory{}
	}

	var defaultMemoryLimit int64 = 256 * 1024 * 1024 // 256MB default worker memory limit
	s.Linux.Resources.Memory.Limit = &defaultMemoryLimit

	return nil
}

func (m *Worker) Start() error {
	m.ContainerID = fmt.Sprintf("%s-%s-%d", m.targetSnapshot, m.ID, rand.Intn(100))
	_, err := m.snapManager.GetRwMounts(m.targetSnapshot, m.ContainerID)
	if err != nil {
		return err
	}

	var processArgs []string
	if m.runtime == "alpine" || m.runtime == "alpinepython" {
		processArgs = []string{m.targetSnapshot}
	} else if m.runtime == "java" {
		processArgs = []string{"/opt/openjdk-13/bin/java", "-cp", ".:gson.jar", "ServerlessFunction"}
	} else {
		processArgs = []string{m.runtime, m.targetSnapshot}
	}

	ipFile, err := ioutil.TempFile("", "container-ip")
	ipFileName := ipFile.Name()
	if err != nil {
		return fmt.Errorf("could not make tmp ip file: %s", err)
	}
	err = ipFile.Close()
	if err != nil {
		return fmt.Errorf("could not close container ip file: %s", err)
	}

	container, err := m.client.NewContainer(
		m.ctx,
		m.ContainerID,
		containerd.WithSnapshot(m.ContainerID),
		containerd.WithNewSpec(WithNetNsHook(ipFileName), oci.WithProcessArgs(processArgs...), WithDefaultMemoryLimit, oci.WithDefaultPathEnv),
	)
	if err != nil {
		return fmt.Errorf("could not create worker container: %s", err)
	}

	m.container = container
	m.connectStdPipes()
	task, err := container.NewTask(m.ctx, m.creator)
	if err != nil {
		return fmt.Errorf("could not create worker task: %s", err)
	}
	m.task = task

	taskExitChan, err := task.Wait(m.ctx)
	if err != nil {
		return fmt.Errorf("could not create worker task channel: %s", err)
	}
	m.taskExitChan = taskExitChan

	err = task.Start(m.ctx)
	if err != nil {
		return fmt.Errorf("could not start worker task: %s", err)
	}

	ipBytes, err := ioutil.ReadFile(ipFileName)
	if err != nil {
		return fmt.Errorf("could not read container ip file: %s", err)
	}

	// fmt.Println(string(ipBytes))
	m.IP = net.ParseIP(string(ipBytes))

	m.controller.SetPid(int(m.task.Pid()))

	return nil
}

func (m *Worker) Activate() error {
	return m.controller.Activate()
}

func (m *Worker) Attach() error {
	return m.controller.Attach()
}

func (m *Worker) TakeCheckpoint() error {
	return m.controller.TakeCheckpoint()
}

func (m *Worker) GetCheckpoints() []*State {
	return m.controller.GetCheckpoints()
}

func (m *Worker) SendFunction(function string) error {
	return m.controller.SendFunction(function)
}

func (m *Worker) SendRequest(request interface{}) (interface{}, error) {
	return m.controller.SendRequest(request)
}

func (m *Worker) AwaitMessage(messageType string) controller.Message {
	return m.controller.AwaitMessage(messageType)
}

func (m *Worker) SendMessage(messageType string, data interface{}) error {
	return m.controller.SendMessage(messageType, data)
}

func (m *Worker) AwaitSignal(waitingFor syscall.Signal) {
	m.controller.AwaitSignal(waitingFor)
}

func (m *Worker) PauseAtSignal(waitingFor syscall.Signal) {
	m.controller.PauseAtSignal(waitingFor)
}

func (m *Worker) Restore() error {
	return m.controller.Restore()
}

func (m *Worker) Detach() error {
	return m.controller.Detach()
}

func (m *Worker) Stop() error {
	return m.controller.Stop()
}

func (m *Worker) Continue() {
	m.controller.ContinueWith(0)
}

func (m *Worker) ContinueWith(signal syscall.Signal) {
	m.controller.ContinueWith(signal)
}

func (m *Worker) ContinueTid(tid int, signal syscall.Signal) {
	m.controller.ContinueTid(tid, signal)
}

func (m *Worker) SendSignalCont(signal syscall.Signal) error {
	return m.controller.SendSignalCont(signal)
}

func (m *Worker) SendSignal(signal syscall.Signal) error {
	return m.controller.SendSignal(signal)
}

func (m *Worker) GetState() (*State, error) {
	return m.controller.GetState()
}

func (m *Worker) GetInitialCheckpoint() (*State, error) {
	return m.controller.GetInitialCheckpoint()
}

func (m *Worker) SetRegs(state *State) error {
	return m.controller.SetRegs(state)
}

func (m *Worker) ClearMemRefs() error {
	return m.controller.ClearMemRefs()
}

func (m *Worker) GetImage(name string) (containerd.Image, error) {
	return m.client.GetImage(m.ctx, name)
}

func (m *Worker) ListImages() ([]containerd.Image, error) {
	return m.client.ListImages(m.ctx)
}

func (m *Worker) Pid() int {
	return int(m.task.Pid())
}

func (m *Worker) End() error {
	var controllerErr error
	if m.task != nil {
		controllerErr = m.controller.End()

		if err := m.task.Kill(m.ctx, syscall.SIGKILL, containerd.WithKillAll); err != nil {
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

	if controllerErr != nil {
		fmt.Errorf("controller failed to end: %s", controllerErr)
	}

	return nil
}

func (m *Worker) CleanSnapshot(name string) error {
	sservice := m.client.SnapshotService("overlayfs")
	return sservice.Remove(m.ctx, name)
}

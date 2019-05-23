package worker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/ostenbom/refunction/worker/ptrace"
	"github.com/ostenbom/refunction/worker/safewriter"
	. "github.com/ostenbom/refunction/worker/state"
	log "github.com/sirupsen/logrus"
)

type Streams struct {
	Stdin  *io.PipeWriter
	Stdout *io.PipeReader
	Stderr *io.PipeReader
}

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
		targetSnapshot: targetSnapshot,
		runtime:        runtime,
		messages:       make(chan Message, 1),
		client:         client,
		ctx:            ctx,
		creator:        cio.NullIO,
		snapManager:    snapManager,
		traceTasks:     make(map[int]*ptrace.TraceTask),
		straceEnabled:  false,
	}, nil
}

type Worker struct {
	ID             string
	ContainerID    string
	targetSnapshot string
	runtime        string
	streams        *Streams
	messages       chan Message
	stderrWriters  []io.Writer
	stdoutWriters  []io.Writer
	client         *containerd.Client
	ctx            context.Context
	creator        cio.Creator
	snapManager    *SnapshotManager
	container      containerd.Container
	task           containerd.Task
	taskExitChan   <-chan containerd.ExitStatus
	checkpoints    []*State
	attached       bool
	attachOptions  []int
	traceTasks     map[int]*ptrace.TraceTask
	straceEnabled  bool
	straceOutput   *safewriter.SafeWriter
	IP             net.IP
}

type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
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

	m.streams = &Streams{
		Stdin:  stdinWrite,
		Stdout: stdoutRead,
		Stderr: stderrRead,
	}

	go func() {
		io.Copy(os.Stderr, stderrRead)
	}()

	go func() {
		// io.Copy(os.Stdout, stdoutRead)
		outBuffer := bufio.NewReader(stdoutRead)

		for {
			line, err := outBuffer.ReadString('\n')
			if err != nil {
				return
			}

			var message Message
			err = json.Unmarshal([]byte(line), &message)
			if err != nil {
				continue
			}

			dataString, ok := message.Data.(string)
			if ok {
				log.Debug(dataString)
			}

			if message.Type == "info" || message.Type == "log" {
				// Ignore these for now
				// fmt.Println(data.Data)
			} else {
				m.messages <- message
			}
		}
	}()
}

func (m *Worker) WithSyscallTrace(to io.Writer) {
	m.straceEnabled = true
	m.attachOptions = append(m.attachOptions, syscall.PTRACE_O_TRACESYSGOOD)
	m.straceOutput = safewriter.NewSafeWriter(to)
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
		processArgs = []string{"/opt/openjdk-13/bin/java", "-cp", ".:json-20180813.jar", "ServerlessFunction"}
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
		containerd.WithNewSpec(WithNetNsHook(ipFileName), oci.WithProcessArgs(processArgs...), WithDefaultMemoryLimit),
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

	m.attached = false

	return nil
}

func (m *Worker) Activate() error {
	m.AwaitMessage("started")
	m.Attach()

	err := m.TakeCheckpoint()
	if err != nil {
		return fmt.Errorf("could not take activation checkpoint: %s", err)
	}

	return nil
}

func (m *Worker) Attach() error {
	taskDirs, err := ioutil.ReadDir(fmt.Sprintf("/proc/%d/task", m.task.Pid()))
	if err != nil {
		return fmt.Errorf("could not read task entries: %s", err)
	}

	for _, t := range taskDirs {
		tid, err := strconv.Atoi(t.Name())
		if err != nil {
			return fmt.Errorf("tid was not int: %s", err)
		}
		task, err := ptrace.NewTraceTask(tid, m.Pid(), m.attachOptions, m.straceEnabled, m.straceOutput)
		if err != nil {
			return fmt.Errorf("could not create trace task %d: %s", tid, err)
		}
		m.traceTasks[tid] = task
	}

	m.attached = true
	return nil
}

func (m *Worker) TakeCheckpoint() error {
	err := m.Stop()
	if err != nil {
		return fmt.Errorf("could not stop for checkpoint")
	}

	checkStart := time.Now()

	state, err := m.GetState()
	if err != nil {
		return err
	}

	err = state.SaveWritablePages()
	if err != nil {
		return err
	}
	err = m.ClearMemRefs()
	if err != nil {
		return err
	}
	m.checkpoints = append(m.checkpoints, state)

	fmt.Printf("checkpoint time: %s", time.Since(checkStart))

	m.Continue()
	return nil
}

func (m *Worker) GetCheckpoints() []*State {
	return m.checkpoints
}

func (m *Worker) SendFunction(function string) error {
	functionReq := &Message{Type: "function", Data: function}
	functionReqString, err := json.Marshal(functionReq)
	if err != nil {
		return err
	}
	newLineReq := append(functionReqString, []byte("\n")...)
	_, err = m.streams.Stdin.Write(newLineReq)
	if err != nil {
		return fmt.Errorf("could not write to worker stdin: %s", err)
	}

	loadedMessage := m.AwaitMessage("function_loaded")
	success, ok := loadedMessage.Data.(bool)
	if !ok || !success {
		return fmt.Errorf("function failed to load")
	}
	return nil
}

func (m *Worker) SendRequest(request interface{}) (interface{}, error) {
	functionReq := &Message{Type: "request", Data: request}
	functionReqString, err := json.Marshal(functionReq)
	if err != nil {
		return "", err
	}
	newLineReq := append(functionReqString, []byte("\n")...)
	_, err = m.streams.Stdin.Write(newLineReq)
	if err != nil {
		return "", err
	}

	message := m.AwaitMessage("response")
	return message.Data, nil
}

func (m *Worker) AwaitMessage(messageType string) Message {
	for {
		message := <-m.messages
		if message.Type == messageType {
			return message
		}
	}
}

// SendMessage writes a message to the containers stdin
func (m *Worker) SendMessage(messageType string, data interface{}) error {
	message := &Message{Type: messageType, Data: data}
	messageString, err := json.Marshal(message)
	if err != nil {
		return err
	}
	newLineReq := append(messageString, []byte("\n")...)
	_, err = m.streams.Stdin.Write(newLineReq)
	if err != nil {
		return err
	}

	return nil
}

// AwaitSignal lets the process continue until the desired signal is caught.
// Allows the process to continue after the signal is caught
func (m *Worker) AwaitSignal(waitingFor syscall.Signal) {
	var waitStat syscall.WaitStatus
	for waitStat.StopSignal() != waitingFor {
		waitStat = <-m.traceTasks[m.Pid()].SignalStop
		m.ContinueWith(waitStat.StopSignal())
	}

	return
}

// PauseAtSignal waits until the desired signal is caught and returns
// before continuing
func (m *Worker) PauseAtSignal(waitingFor syscall.Signal) {
	var waitStat syscall.WaitStatus
	waitStat = <-m.traceTasks[m.Pid()].SignalStop

	for waitStat.StopSignal() != waitingFor {
		m.ContinueWith(waitStat.StopSignal())
		waitStat = <-m.traceTasks[m.Pid()].SignalStop
	}

	m.traceTasks[m.Pid()].SignalStop <- waitStat
	return
}

// Restore returns process state to first checkpoint
// Restore takes responsibility for stopping tasks
func (m *Worker) Restore() error {
	err := m.Stop()
	if err != nil {
		return fmt.Errorf("could not stop worker for restore: %s", err)
	}

	if len(m.checkpoints) <= 0 {
		return fmt.Errorf("no checkpoints to restore")
	}

	state := m.checkpoints[0]

	start := time.Now()

	err = state.RestoreDirtyPages()
	if err != nil {
		return fmt.Errorf("could not restore stack: %s", err)
	}
	err = state.RestoreRegs()
	if err != nil {
		return fmt.Errorf("could not restore regs: %s", err)
	}
	fmt.Printf("restore time: %s", time.Since(start))

	m.Continue()

	return nil
}

// Detach 'es all tasks from ptrace supervision
// For PTRACE_DETACH to on a task, it must be in a ptrace-stop state
// Tgkilling the task and supressing injection on detach is a good way to
// do this.
func (m *Worker) Detach() error {
	for _, task := range m.traceTasks {
		// Ensure the task is stopped
		err := task.Stop()
		if err != nil {
			return fmt.Errorf("could not stop child for detach: %s", err)
		}

		task.Detach <- 1
		<-task.HasDetached
	}
	m.attached = false
	return nil
}

func (m *Worker) Stop() error {
	for _, t := range m.traceTasks {
		err := t.Stop()
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Worker) Continue() {
	m.ContinueWith(0)
}

func (m *Worker) ContinueWith(signal syscall.Signal) {
	for tid := range m.traceTasks {
		m.ContinueTid(tid, signal)
	}
}

func (m *Worker) ContinueTid(tid int, signal syscall.Signal) {
	m.traceTasks[tid].Continue <- signal
	<-m.traceTasks[tid].HasContinued
}

func (m *Worker) SendSignalCont(signal syscall.Signal) error {
	err := m.SendSignal(signal)
	if err != nil {
		return err
	}

	// If not attached, signal will go through
	if !m.attached {
		return nil
	}

	<-m.traceTasks[m.Pid()].SignalStop
	m.ContinueWith(signal)
	return nil
}

func (m *Worker) SendSignal(signal syscall.Signal) error {
	pid := m.Pid()
	return syscall.Tgkill(pid, pid, signal)
}

// GetState creates a new instance of the process state.
// Caller must ensure tasks are stopped
func (m *Worker) GetState() (*State, error) {
	state, err := NewState(m.Pid(), m.traceTasks)
	if err != nil {
		return nil, fmt.Errorf("could not get state: %s", err)
	}

	return state, nil
}

// SetRegs returns registers to their values in state
// Caller must ensure tasks are stopped
func (m *Worker) SetRegs(state *State) error {
	err := state.RestoreRegs()
	if err != nil {
		return fmt.Errorf("could not set regs: %s", err)
	}

	return nil
}

func (m *Worker) ClearMemRefs() error {
	pid := m.Pid()
	f, err := os.OpenFile(fmt.Sprintf("/proc/%d/clear_refs", pid), os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("could not open clear_refs for pid %d: %s", pid, err)
	}
	defer f.Close()

	_, err = f.WriteString("4")
	if err != nil {
		return fmt.Errorf("could not clear_refs for pid %d: %s", pid, err)
	}
	return nil
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
	var detachErr error
	if m.task != nil {
		if m.attached {
			detachErr = m.Detach()
		}
		if m.streams != nil {
			m.streams.Stdin.Close()
			m.streams.Stdout.Close()
			m.streams.Stderr.Close()
		}
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

	if detachErr != nil {
		return fmt.Errorf("could not detach on end: %s", detachErr)
	}

	return nil
}

func (m *Worker) CleanSnapshot(name string) error {
	sservice := m.client.SnapshotService("overlayfs")
	return sservice.Remove(m.ctx, name)
}

package worker

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	. "github.com/ostenbom/refunction/worker/state"
)

const RuntimeStartedSignal = syscall.SIGUSR2
const ActivateChildSignal = syscall.SIGUSR1
const CheckpointSignal = syscall.SIGUSR1
const FinishServerSignal = syscall.SIGUSR2
const ServerFinishedSignal = syscall.SIGUSR2

type PtraceChannels struct {
	SignalStop     chan syscall.WaitStatus
	Continue       chan syscall.Signal
	HasContinued   chan int
	Detach         chan int
	HasDetached    chan int
	InStopFunction chan func()
	Error          chan error
}

type CommunicationType int

const (
	SocketCommunication CommunicationType = iota + 1
	StdPipeCommunication
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

	return &Worker{
		ID:                     id,
		targetSnapshot:         targetSnapshot,
		runtime:                runtime,
		communication:          SocketCommunication,
		responses:              make(chan string, 1),
		functionLoadedMessages: make(chan string, 1),
		client:                 client,
		ctx:                    ctx,
		creator:                cio.NullIO,
		snapManager:            snapManager,
		ptrace: PtraceChannels{
			SignalStop:     make(chan syscall.WaitStatus, 1),
			Continue:       make(chan syscall.Signal),
			HasContinued:   make(chan int),
			Detach:         make(chan int),
			HasDetached:    make(chan int),
			InStopFunction: make(chan func()),
			Error:          make(chan error),
		},
		straceEnabled: false,
	}, nil
}

type Worker struct {
	ID                     string
	ContainerID            string
	targetSnapshot         string
	runtime                string
	communication          CommunicationType
	streams                *Streams
	responses              chan string
	functionLoadedMessages chan string
	client                 *containerd.Client
	ctx                    context.Context
	creator                cio.Creator
	snapManager            *SnapshotManager
	container              containerd.Container
	task                   containerd.Task
	taskExitChan           <-chan containerd.ExitStatus
	checkpoints            []*State
	attached               bool
	attachOptions          []int
	ptrace                 PtraceChannels
	straceEnabled          bool
	straceOutput           io.Writer
	IP                     net.IP
}

type LoadFunctionReq struct {
	Type    string `json:"type"`
	Handler string `json:"handler"`
}

type FunctionData struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

type RequestResponse struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

func (m *Worker) WithStdPipeCommunicationExtras(stderrWriter io.Writer, stdoutWriters ...io.Writer) {
	m.withStdPipeCommunication([]io.Writer{stderrWriter}, stdoutWriters)
}

func (m *Worker) WithStdPipeCommunication() {
	m.withStdPipeCommunication([]io.Writer{}, []io.Writer{})
}

func (m *Worker) withStdPipeCommunication(stderrWriters []io.Writer, stdoutWriters []io.Writer) {
	m.communication = StdPipeCommunication
	stdinRead, stdinWrite := io.Pipe()
	stdoutRead, stdoutWrite := io.Pipe()
	stderrRead, stderrWrite := io.Pipe()

	var collectedStdErr io.Writer
	if len(stderrWriters) > 0 {
		collectedStdErr = io.MultiWriter(append(stderrWriters, stderrWrite)...)
	} else {
		collectedStdErr = stderrWrite
	}

	var collectedStdOut io.Writer
	if len(stderrWriters) > 0 {
		collectedStdOut = io.MultiWriter(append(stdoutWriters, stdoutWrite)...)
	} else {
		collectedStdOut = stderrWrite
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

			var data RequestResponse
			err = json.Unmarshal([]byte(line), &data)
			if err != nil {
				continue
			}

			// switch v := data.Data.(type) {
			// case string:
			// 	fmt.Printf("Data is: %s\n", v)
			// default:
			// 	fmt.Printf("Data was something else: %s\n", reflect.TypeOf(data.Data))
			// }

			if data.Type == "info" {
				// fmt.Println(data.Data)
			} else if data.Type == "response" {
				m.responses <- string(data.Data)
			} else if data.Type == "function_loaded" {
				m.functionLoadedMessages <- "loaded"
			}
		}
	}()
}

func (m *Worker) WithCreator(creator cio.Creator) {
	m.creator = creator
}

func (m *Worker) WithSyscallTrace(to io.Writer) {
	m.straceEnabled = true
	m.attachOptions = append(m.attachOptions, syscall.PTRACE_O_TRACESYSGOOD)
	m.straceOutput = to
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

func (m *Worker) Start() error {
	err := m.snapManager.CreateLayerFromBase(m.targetSnapshot)
	if err != nil {
		return err
	}

	m.ContainerID = fmt.Sprintf("%s-%s-%d", m.targetSnapshot, m.ID, rand.Intn(100))
	_, err = m.snapManager.GetRwMounts(m.targetSnapshot, m.ContainerID)
	if err != nil {
		return err
	}

	var processArgs []string
	if m.runtime == "alpine" || m.runtime == "alpinepython" {
		processArgs = []string{m.targetSnapshot}
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
		containerd.WithNewSpec(WithNetNsHook(ipFileName), oci.WithProcessArgs(processArgs...)),
	)
	if err != nil {
		return fmt.Errorf("could not create worker container: %s", err)
	}

	m.container = container

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
	m.Attach()
	m.Continue()
	m.AwaitSignal(RuntimeStartedSignal)

	err := m.SendSignal(ActivateChildSignal)
	if err != nil {
		return fmt.Errorf("could not send activate signal: %s", err)
	}
	err = m.TakeCheckpoint()
	if err != nil {
		return fmt.Errorf("could not take activation checkpoint: %s", err)
	}

	return nil
}

func (m *Worker) TakeCheckpoint() error {
	m.PauseAtSignal(CheckpointSignal)

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

	m.ContinueWith(CheckpointSignal)
	return nil
}

func (m *Worker) GetCheckpoints() []*State {
	return m.checkpoints
}

func (m *Worker) SendFunction(function string) error {
	if m.communication == StdPipeCommunication {
		functionReq := &LoadFunctionReq{Type: "function", Handler: function}
		functionReqString, err := json.Marshal(functionReq)
		if err != nil {
			return err
		}
		newLineReq := append(functionReqString, []byte("\n")...)
		_, err = m.streams.Stdin.Write(newLineReq)

		<-m.functionLoadedMessages
		return err
	}

	tcpAddr := net.TCPAddr{
		IP:   m.IP,
		Port: 5000,
	}

	functionReq := &LoadFunctionReq{Handler: function}
	functionReqString, err := json.Marshal(functionReq)
	if err != nil {
		return fmt.Errorf("could not marshal function: %s, %s", function, err)
	}

	conn, err := net.DialTCP("tcp", nil, &tcpAddr)
	if err != nil {
		return fmt.Errorf("could not dial worker: %s", err)
	}
	defer conn.Close()

	_, err = conn.Write(functionReqString)
	if err != nil {
		return fmt.Errorf("could not write to worker: %s", err)
	}

	return nil
}

func (m *Worker) SendRequest(request string) (string, error) {
	if m.communication == StdPipeCommunication {
		functionReq := &FunctionData{Type: "request", Data: request}
		functionReqString, err := json.Marshal(functionReq)
		if err != nil {
			return "", err
		}
		newLineReq := append(functionReqString, []byte("\n")...)
		_, err = m.streams.Stdin.Write(newLineReq)
		if err != nil {
			return "", err
		}

		return <-m.responses, nil
	}

	tcpAddr := net.TCPAddr{
		IP:   m.IP,
		Port: 5000,
	}

	conn, err := net.DialTCP("tcp", nil, &tcpAddr)
	if err != nil {
		return "", fmt.Errorf("could not dial worker: %s", err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte(request))
	if err != nil {
		return "", fmt.Errorf("could not write to worker: %s", err)
	}

	decoder := json.NewDecoder(conn)
	var resp RequestResponse

	err = decoder.Decode(&resp)
	if err != nil {
		return "", fmt.Errorf("could not get request response: %s", err)
	}

	return string(resp.Data), nil
}

// AwaitSignal lets the process continue until the desired signal is caught.
// Allows the process to continue after the signal is caught
func (m *Worker) AwaitSignal(waitingFor syscall.Signal) {
	var waitStat syscall.WaitStatus
	for waitStat.StopSignal() != waitingFor {
		waitStat = <-m.ptrace.SignalStop
		m.ContinueWith(waitStat.StopSignal())
	}

	return
}

// PauseAtSignal waits until the desired signal is caught and returns
// before continuing
func (m *Worker) PauseAtSignal(waitingFor syscall.Signal) {
	var waitStat syscall.WaitStatus
	waitStat = <-m.ptrace.SignalStop

	for waitStat.StopSignal() != waitingFor {
		m.ContinueWith(waitStat.StopSignal())
		waitStat = <-m.ptrace.SignalStop
	}

	m.ptrace.SignalStop <- waitStat
	return
}

func (m *Worker) FinishFunction() error {
	err := m.SendSignal(syscall.SIGUSR2)
	if err != nil {
		return fmt.Errorf("could not finish function: %s", err)
	}
	m.AwaitSignal(syscall.SIGUSR2)
	return nil
}

func (m *Worker) Restore() error {
	var stoppedByRestore bool
	select {
	case <-m.ptrace.SignalStop:
		stoppedByRestore = false
	default:
		stoppedByRestore = true
	}

	if stoppedByRestore {
		err := m.Stop()
		if err != nil {
			return fmt.Errorf("could not stop worker for restore: %s", err)
		}
	}

	if len(m.checkpoints) <= 0 {
		return fmt.Errorf("no checkpoints to restore")
	}

	state := m.checkpoints[0]

	start := time.Now()

	err := state.RestoreDirtyPages()
	if err != nil {
		return fmt.Errorf("could not restore stack: %s", err)
	}
	err = state.RestoreRegs()
	if err != nil {
		return fmt.Errorf("could not restore regs: %s", err)
	}
	fmt.Printf("restore time: %s", time.Since(start))

	if stoppedByRestore {
		m.Continue()
	}

	return nil
}

func (m *Worker) Detach() error {
	var err error
	select {
	case <-m.ptrace.SignalStop:
		err = nil
		break
	default:
		err = m.Stop()
	}
	if err != nil {
		return fmt.Errorf("could not stop child for detach: %s", err)
	}

	m.attached = false
	m.ptrace.Detach <- 1
	<-m.ptrace.HasDetached
	return nil
}

func (m *Worker) Stop() error {
	err := syscall.Kill(int(m.task.Pid()), syscall.SIGSTOP)
	if err != nil {
		return err
	}

	stop := <-m.ptrace.SignalStop
	m.ptrace.SignalStop <- stop
	return err
}

func (m *Worker) Continue() {
	m.ContinueWith(0)
}

func (m *Worker) ContinueWith(signal syscall.Signal) {
	m.ptrace.Continue <- signal
	<-m.ptrace.HasContinued
}

func (m *Worker) SendSignal(signal syscall.Signal) error {
	pid := int(m.task.Pid())

	err := syscall.Kill(pid, signal)
	if err != nil {
		return err
	}

	// If not attached, signal will go through
	if !m.attached {
		return nil
	}

	<-m.ptrace.SignalStop
	m.ContinueWith(signal)
	return nil
}

func (m *Worker) GetState() (*State, error) {
	select {
	case wait := <-m.ptrace.SignalStop:
		m.ptrace.SignalStop <- wait
		break
	default:
		err := m.Stop()
		if err != nil {
			return nil, fmt.Errorf("could not stop child to get state: %s", err)
		}
		defer m.Continue()
	}

	state, err := NewState(int(m.task.Pid()), m.ptrace.InStopFunction)
	if err != nil {
		return nil, fmt.Errorf("could not get state: %s", err)
	}

	return state, nil
}

func (m *Worker) SetRegs(state *State) error {
	select {
	case wait := <-m.ptrace.SignalStop:
		m.ptrace.SignalStop <- wait
		break
	default:
		err := m.Stop()
		if err != nil {
			return fmt.Errorf("could not stop child to set regs: %s", err)
		}
		defer m.Continue()
	}

	err := state.RestoreRegs()
	if err != nil {
		return fmt.Errorf("could not set regs: %s", err)
	}

	return nil
}

func (m *Worker) ClearMemRefs() error {
	pid := int(m.task.Pid())
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

func (m *Worker) Pid() (uint32, error) {
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
	// TODO
	// err := m.Stop()
	// if err != nil {
	// 	return fmt.Errorf("could not stop child for regs print: %s", err)
	// }
	// defer m.Continue()
	//
	// var regs syscall.PtraceRegs
	// err = syscall.PtraceGetRegs(int(m.task.Pid()), &regs)
	// if err != nil {
	// 	return fmt.Errorf("could not get regs: %s", err)
	// }
	//
	// fmt.Printf("Regs: %+v\n", regs)

	return nil
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

	if detachErr != nil {
		return fmt.Errorf("could not detach on end: %s", detachErr)
	}

	return nil
}

func (m *Worker) CleanSnapshot(name string) error {
	sservice := m.client.SnapshotService("overlayfs")
	return sservice.Remove(m.ctx, name)
}

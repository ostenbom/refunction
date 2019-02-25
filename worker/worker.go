package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"runtime"
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

type ChildState int

const (
	Running ChildState = iota + 1
	SignalStop
	SyscallStop
	EventStop
	Exited
	// SeccompStop
)

func (state ChildState) String() string {
	names := []string{
		"Running",
		"SignalStop",
		"SyscallStop",
		"EventStop",
		"Exited",
	}

	return names[state]
}

func (state ChildState) IsStopped() bool {
	return state != Running
}

type WaitChannels struct {
	SignalStop  chan syscall.WaitStatus
	SyscallStop chan syscall.WaitStatus
	EventStop   chan syscall.WaitStatus
}

func NewWorker(id string, client *containerd.Client, runtime, targetSnapshot string) (*Worker, error) {
	ctx := namespaces.WithNamespace(context.Background(), "refunction-worker"+id)

	snapManager, err := NewSnapshotManager(ctx, client, runtime)
	if err != nil {
		return nil, err
	}

	return &Worker{
		ID:             id,
		targetSnapshot: targetSnapshot,
		runtime:        runtime,
		client:         client,
		ctx:            ctx,
		creator:        cio.NullIO,
		snapManager:    snapManager,
		waitChannels: WaitChannels{
			SignalStop:  make(chan syscall.WaitStatus),
			SyscallStop: make(chan syscall.WaitStatus),
			EventStop:   make(chan syscall.WaitStatus),
		},
		straceEnabled: false,
	}, nil
}

type Worker struct {
	ID             string
	ContainerID    string
	targetSnapshot string
	runtime        string
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
	state          ChildState
	waitChannels   WaitChannels
	straceEnabled  bool
	IP             net.IP
}

type LoadFunctionReq struct {
	Handler string `json:"handler"`
}

func (m *Worker) WithCreator(creator cio.Creator) {
	m.creator = creator
}

func (m *Worker) WithSyscallTrace() {
	m.straceEnabled = true
	m.attachOptions = append(m.attachOptions, syscall.PTRACE_O_TRACESYSGOOD)
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
	if m.runtime != "alpine" {
		processArgs = []string{m.runtime, m.targetSnapshot}
	} else {
		processArgs = []string{m.targetSnapshot}
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

	m.IP = net.ParseIP(string(ipBytes))

	m.attached = false
	m.state = Running

	return nil
}

func (m *Worker) Activate() error {
	err := m.Attach()
	if err != nil {
		return fmt.Errorf("could not attach activate: %s", err)
	}
	err = m.Continue()
	if err != nil {
		return fmt.Errorf("could not continue to activate: %s", err)
	}
	err = m.AwaitSignal(RuntimeStartedSignal)
	if err != nil {
		return fmt.Errorf("could not await runtime started in activate: %s", err)
	}
	err = m.SendSignal(ActivateChildSignal)
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
	err := m.PauseAtSignal(CheckpointSignal)
	if err != nil {
		return err
	}

	checkStart := time.Now()

	state, err := m.GetState()
	if err != nil {
		return err
	}

	err = state.SavePages("[stack]")
	if err != nil {
		return err
	}
	err = state.SavePages("[heap]")
	if err != nil {
		return err
	}
	err = m.ClearMemRefs()
	if err != nil {
		return err
	}
	m.checkpoints = append(m.checkpoints, state)

	fmt.Printf("checkpoint time: %s", time.Since(checkStart))

	return m.ContinueWith(CheckpointSignal)
}

func (m *Worker) GetCheckpoints() []*State {
	return m.checkpoints
}

func (m *Worker) SendFunction(function string) error {
	udpAddr := net.TCPAddr{
		IP:   m.IP,
		Port: 5000,
	}

	functionReq := &LoadFunctionReq{Handler: function}
	functionReqString, err := json.Marshal(functionReq)
	if err != nil {
		return fmt.Errorf("could not marshal function: %s, %s", function, err)
	}

	conn, err := net.DialTCP("tcp", nil, &udpAddr)
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
	udpAddr := net.TCPAddr{
		IP:   m.IP,
		Port: 5000,
	}

	conn, err := net.DialTCP("tcp", nil, &udpAddr)
	if err != nil {
		return "", fmt.Errorf("could not dial worker: %s", err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte(request))
	if err != nil {
		return "", fmt.Errorf("could not write to worker: %s", err)
	}

	response, err := ioutil.ReadAll(conn)
	if err != nil {
		return "", fmt.Errorf("could not get request response: %s", err)
	}

	return string(response), nil
}

// AwaitSignal lets the process continue until the desired signal is caught.
// Allows the process to continue after the signal is caught
func (m *Worker) AwaitSignal(waitingFor syscall.Signal) error {

	var waitStat syscall.WaitStatus
	for waitStat.StopSignal() != waitingFor {
		waitStat = <-m.waitChannels.SignalStop

		err := m.ContinueWith(waitStat.StopSignal())
		if err != nil {
			return err
		}
	}

	return nil
}

// PauseAtSignal waits until the desired signal is caught and returns
// before continuing
func (m *Worker) PauseAtSignal(waitingFor syscall.Signal) error {
	var waitStat syscall.WaitStatus
	waitStat = <-m.waitChannels.SignalStop

	for waitStat.StopSignal() != waitingFor {
		err := m.ContinueWith(waitStat.StopSignal())
		if err != nil {
			return err
		}

		waitStat = <-m.waitChannels.SignalStop
	}

	return nil
}

func (m *Worker) Restore() error {
	stoppedByRestore := false
	if !m.state.IsStopped() {
		stoppedByRestore = true
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

	err := state.RestoreDirtyPages("[stack]")
	if err != nil {
		return fmt.Errorf("could not restore stack: %s", err)
	}
	err = state.RestoreDirtyPages("[heap]")
	if err != nil {
		return fmt.Errorf("could not restore heap: %s", err)
	}
	err = state.RestoreRegs()
	if err != nil {
		return fmt.Errorf("could not restore regs: %s", err)
	}
	fmt.Printf("restore time: %s", time.Since(start))

	if stoppedByRestore {
		err = m.Continue()
		if err != nil {
			return fmt.Errorf("could not continue after restore: %s", err)
		}
	}

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
	m.state = SignalStop

	_, err = syscall.Wait4(int(m.task.Pid()), nil, 0, nil)
	if err != nil {
		return err
	}

	for _, opt := range m.attachOptions {
		err := syscall.PtraceSetOptions(int(m.task.Pid()), opt)
		if err != nil {
			return fmt.Errorf("could not set ptrace option on attach: %s", err)
		}
	}

	m.beginWaitLoop()

	return nil
}

func (m *Worker) beginWaitLoop() {
	go func() {
		var waitStat syscall.WaitStatus
		for m.state != Exited {
			_, err := syscall.Wait4(int(m.task.Pid()), &waitStat, 0, nil)
			if err != nil {
				if waitStat.Exited() {
					break
				}

				fmt.Printf("error in waiting for child loop: %s", err)
				// Alternative to panic, better to end the worker at this point
				m.End()
			}

			if !waitStat.Stopped() {
				fmt.Println("did not wait for a stopped signal!")
				continue
			}

			if waitStat.StopSignal() == syscall.SIGTRAP {
				fmt.Printf("it was a trap signal, so syscall or event: %d", waitStat)
				fmt.Printf("trap cause gives me: %d", waitStat.TrapCause())
				fmt.Printf("compared as per man7: %t", waitStat.TrapCause() == int(syscall.SIGTRAP|0x80))
			} else {
				m.state = SignalStop
				m.waitChannels.SignalStop <- waitStat
			}
		}
	}()
}

func (m *Worker) Detach() error {
	if !m.state.IsStopped() {
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
	m.state = Running
	runtime.UnlockOSThread()
	return nil
}

func (m *Worker) Stop() error {
	err := syscall.Kill(int(m.task.Pid()), syscall.SIGSTOP)
	if err != nil {
		return err
	}

	<-m.waitChannels.SignalStop
	return err
}

func (m *Worker) Continue() error {
	return m.ContinueWith(0)
}

func (m *Worker) ContinueWith(signal syscall.Signal) error {
	m.state = Running
	return syscall.PtraceCont(int(m.task.Pid()), int(signal))
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

	var waitStat syscall.WaitStatus
	waitStat = <-m.waitChannels.SignalStop

	return m.ContinueWith(waitStat.StopSignal())
}

func (m *Worker) GetState() (*State, error) {
	if !m.state.IsStopped() {
		err := m.Stop()
		if err != nil {
			return nil, fmt.Errorf("could not stop child to get state: %s", err)
		}
		defer m.Continue()
	}

	state, err := NewState(int(m.task.Pid()))
	if err != nil {
		return nil, fmt.Errorf("could not get state: %s", err)
	}

	return state, nil
}

func (m *Worker) SetRegs(state *State) error {
	if !m.state.IsStopped() {
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
	var detachErr error
	if m.task != nil {
		if m.attached {
			detachErr = m.Detach()
		}
		if err := m.task.Kill(m.ctx, syscall.SIGKILL); err != nil {
			if errdefs.IsFailedPrecondition(err) || errdefs.IsNotFound(err) {
				return nil
			}
			return fmt.Errorf("failed to kill in manager end: %s", err)
		}

		<-m.taskExitChan
		m.state = Exited

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

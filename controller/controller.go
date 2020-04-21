package controller

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/ostenbom/refunction/controller/ptrace"
	"github.com/ostenbom/refunction/controller/safewriter"
	"github.com/ostenbom/refunction/state"
	"github.com/prometheus/common/log"
)

type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type Streams struct {
	Stdin  *io.PipeWriter
	Stdout *io.PipeReader
	Stderr *io.PipeReader
}

type Controller struct {
	pid           int
	messages      chan Message
	streams       *Streams
	traceTasks    map[int]*ptrace.TraceTask
	checkpoints   []*state.State
	attached      bool
	ptraceOptions ptrace.Options
}

func NewController() *Controller {
	return &Controller{
		attached:   false,
		messages:   make(chan Message, 1),
		traceTasks: make(map[int]*ptrace.TraceTask),
		ptraceOptions: ptrace.Options{
			StraceEnabled: false,
		},
	}
}

func (c *Controller) WithSyscallTrace(to io.Writer) {
	c.ptraceOptions.StraceEnabled = true
	c.ptraceOptions.AttachOptions = append(c.ptraceOptions.AttachOptions, syscall.PTRACE_O_TRACESYSGOOD)
	c.ptraceOptions.StraceOutput = safewriter.NewSafeWriter(to)
}

func (c *Controller) SetStreams(in *io.PipeWriter, out *io.PipeReader, err *io.PipeReader) {
	c.streams = &Streams{
		Stdin:  in,
		Stdout: out,
		Stderr: err,
	}

	go func() {
		// Uncomment for debugging
		// io.Copy(os.Stdout, stdoutRead)
		outBuffer := bufio.NewReader(out)

		for {
			line, err := outBuffer.ReadString('\n')
			if err != nil {
				return
			}

			var message Message
			err = json.Unmarshal([]byte(line), &message)
			if err != nil {
				log.Debug(line)
				continue
			}

			dataString, ok := message.Data.(string)
			if ok {
				log.Debug(dataString)
			}

			if message.Type == "info" || message.Type == "log" {
				// Ignore these for now
				// log.Debug(message.Data)
			} else {
				c.messages <- message
			}
		}
	}()
}

func (c *Controller) SetPid(pid int) {
	c.pid = pid
}

func (c *Controller) Activate() error {
	c.AwaitMessage("started")

	err := c.Attach()
	if err != nil {
		return fmt.Errorf("could not attach to process: %s", err)
	}

	err = c.TakeCheckpoint()
	if err != nil {
		return fmt.Errorf("could not take activation checkpoint: %s", err)
	}

	return nil
}

func (c *Controller) Attach() error {
	// TODO if pid is 0 check
	taskDirs, err := ioutil.ReadDir(fmt.Sprintf("/proc/%d/task", c.pid))
	if err != nil {
		return fmt.Errorf("could not read task entries: %s", err)
	}

	for _, t := range taskDirs {
		tid, err := strconv.Atoi(t.Name())
		if err != nil {
			return fmt.Errorf("tid was not int: %s", err)
		}

		task, err := ptrace.NewTraceTask(tid, c.pid, c.ptraceOptions)
		if err != nil {
			return fmt.Errorf("could not create trace task %d: %s", tid, err)
		}

		c.traceTasks[tid] = task
	}

	c.attached = true
	return nil
}

func (c *Controller) TakeCheckpoint() error {
	err := c.Stop()
	if err != nil {
		return fmt.Errorf("could not stop for checkpoint")
	}

	checkStart := time.Now()

	state, err := c.GetState()
	if err != nil {
		return err
	}

	err = state.SaveWritablePages()
	if err != nil {
		return err
	}
	err = c.ClearMemRefs()
	if err != nil {
		return err
	}
	c.checkpoints = append(c.checkpoints, state)

	fmt.Printf("checkpoint time: %s", time.Since(checkStart))

	c.Continue()
	return nil
}

func (c *Controller) GetCheckpoints() []*state.State {
	return c.checkpoints
}

func (c *Controller) SendFunction(function string) error {
	functionReq := &Message{Type: "function", Data: function}

	functionReqString, err := json.Marshal(functionReq)
	if err != nil {
		return err
	}

	newLineReq := append(functionReqString, []byte("\n")...)
	_, err = c.streams.Stdin.Write(newLineReq)
	if err != nil {
		return fmt.Errorf("could not write to worker stdin: %s", err)
	}

	loadedMessage := c.AwaitMessage("function_loaded")
	success, ok := loadedMessage.Data.(bool)
	if !ok || !success {
		return fmt.Errorf("function failed to load")
	}

	return nil
}

func (c *Controller) SendRequest(request interface{}) (interface{}, error) {
	functionReq := &Message{Type: "request", Data: request}
	functionReqString, err := json.Marshal(functionReq)
	if err != nil {
		return "", err
	}
	newLineReq := append(functionReqString, []byte("\n")...)
	_, err = c.streams.Stdin.Write(newLineReq)
	if err != nil {
		return "", err
	}

	message := c.AwaitMessage("response")
	return message.Data, nil
}

func (c *Controller) AwaitMessage(messageType string) Message {
	for {
		message := <-c.messages
		if message.Type == messageType {
			return message
		}
	}
}

// SendMessage writes a message to the containers stdin
func (c *Controller) SendMessage(messageType string, data interface{}) error {
	message := &Message{Type: messageType, Data: data}
	messageString, err := json.Marshal(message)
	if err != nil {
		return err
	}
	newLineReq := append(messageString, []byte("\n")...)
	_, err = c.streams.Stdin.Write(newLineReq)
	if err != nil {
		return err
	}

	return nil
}

// AwaitSignal lets the process continue until the desired signal is caught.
// Allows the process to continue after the signal is caught
func (c *Controller) AwaitSignal(waitingFor syscall.Signal) {
	var waitStat syscall.WaitStatus
	for waitStat.StopSignal() != waitingFor {
		waitStat = <-c.traceTasks[c.pid].SignalStop
		c.ContinueWith(waitStat.StopSignal())
	}
}

// PauseAtSignal waits until the desired signal is caught and returns
// before continuing
func (c *Controller) PauseAtSignal(waitingFor syscall.Signal) {
	var waitStat syscall.WaitStatus
	waitStat = <-c.traceTasks[c.pid].SignalStop

	for waitStat.StopSignal() != waitingFor {
		c.ContinueWith(waitStat.StopSignal())
		waitStat = <-c.traceTasks[c.pid].SignalStop
	}

	c.traceTasks[c.pid].SignalStop <- waitStat
}

// Restore returns process state to first checkpoint
// Restore takes responsibility for stopping tasks
func (c *Controller) Restore() error {
	err := c.Stop()
	if err != nil {
		return fmt.Errorf("could not stop worker for restore: %s", err)
	}

	if len(c.checkpoints) == 0 {
		return fmt.Errorf("no checkpoints to restore")
	}

	state := c.checkpoints[0]

	start := time.Now()

	fixup := false
	changed, err := state.ProgramBreakChanged()
	if err != nil {
		return fmt.Errorf("could not check program break on restore: %s", err)
	}
	if changed {
		fixup = true
		err := state.RestoreProgramBreak()
		if err != nil {
			return fmt.Errorf("count not restore program break: %s", err)
		}
	}

	changed, err = state.NumMemoryLocationsChanged()
	if err != nil {
		return fmt.Errorf("could not check num mem locations changed on restore: %s", err)
	}
	if changed {
		fixup = true
		err := state.UnmapNewLocations()
		if err != nil {
			return fmt.Errorf("count not unmap new locations: %s", err)
		}
	}

	if fixup {
		err := state.FixupSyscallState()
		if err != nil {
			return fmt.Errorf("count not fixup syscall state: %s", err)
		}
	}

	err = state.RestoreDirtyPages()
	if err != nil {
		return fmt.Errorf("could not restore stack: %s", err)
	}
	err = state.RestoreRegs()
	if err != nil {
		return fmt.Errorf("could not restore regs: %s", err)
	}
	fmt.Printf("restore time: %s", time.Since(start))

	c.Continue()

	return nil
}

// Detach 'es all tasks from ptrace supervision
// For PTRACE_DETACH to on a task, it must be in a ptrace-stop state
// Tgkilling the task and supressing injection on detach is a good way to
// do this.
func (c *Controller) Detach() error {
	for _, task := range c.traceTasks {
		// Ensure the task is stopped
		err := task.Stop()
		if err != nil {
			return fmt.Errorf("could not stop child for detach: %s", err)
		}

		task.Detach <- 1
		<-task.HasDetached
	}

	c.attached = false

	return nil
}

func (c *Controller) Stop() error {
	for _, t := range c.traceTasks {
		err := t.Stop()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) Continue() {
	c.ContinueWith(0)
}

func (c *Controller) ContinueWith(signal syscall.Signal) {
	for tid := range c.traceTasks {
		c.ContinueTid(tid, signal)
	}
}

func (c *Controller) ContinueTid(tid int, signal syscall.Signal) {
	c.traceTasks[tid].Continue <- signal
	<-c.traceTasks[tid].HasContinued
}

func (c *Controller) SendSignalCont(signal syscall.Signal) error {
	err := c.SendSignal(signal)
	if err != nil {
		return err
	}

	// If not attached, signal will go through
	if !c.attached {
		return nil
	}

	<-c.traceTasks[c.pid].SignalStop
	c.ContinueWith(signal)
	return nil
}

func (c *Controller) SendSignal(signal syscall.Signal) error {
	pid := c.pid
	return syscall.Tgkill(pid, pid, signal)
}

// GetState creates a new instance of the process state.
// Caller must ensure tasks are stopped
func (c *Controller) GetState() (*state.State, error) {
	state, err := state.NewState(c.pid, c.traceTasks)
	if err != nil {
		return nil, fmt.Errorf("could not get state: %s", err)
	}

	return state, nil
}

func (c *Controller) GetInitialCheckpoint() (*state.State, error) {
	if len(c.checkpoints) == 0 {
		return nil, fmt.Errorf("no initial checkpoint")
	}
	return c.checkpoints[0], nil
}

// SetRegs returns registers to their values in state
// Caller must ensure tasks are stopped
func (c *Controller) SetRegs(state *state.State) error {
	err := state.RestoreRegs()
	if err != nil {
		return fmt.Errorf("could not set regs: %s", err)
	}

	return nil
}

func (c *Controller) ClearMemRefs() error {
	pid := c.pid
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

func (c *Controller) End() error {
	var detachErr error
	if c.attached {
		detachErr = c.Detach()
	}
	if c.streams != nil {
		c.streams.Stdin.Close()
		c.streams.Stdout.Close()
		c.streams.Stderr.Close()
	}

	if detachErr != nil {
		return fmt.Errorf("could not detach on end: %s", detachErr)
	}

	return nil
}

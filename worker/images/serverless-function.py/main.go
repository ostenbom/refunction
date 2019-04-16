package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"syscall"

	. "github.com/ostenbom/refunction/worker"
)

func main() {
	runtime.LockOSThread()
	if len(os.Args) <= 1 {
		panic("provide pid please")
	}
	pidString := os.Args[1]
	pid, err := strconv.Atoi(pidString)
	if err != nil {
		panic(err)
	}

	worker := NewLocalWorker("local-serverless-function", uint32(pid))
	wPid, _ := worker.Pid()

	worker.WithSyscallTrace(os.Stdout)

	fmt.Printf("pid taken as %s, %d\n", pidString, wPid)

	err = worker.Activate()
	if err != nil {
		panic(err)
	}

	fmt.Println("activated")

	buf := bufio.NewReader(os.Stdin)
	fmt.Print("> ")
	_, _ = buf.ReadBytes('\n')

	function := "def handle(req):\n  print(req)\n  return req"
	err = worker.SendFunction(function)
	if err != nil {
		panic(err)
	}

	request := "{\"greatkey\": \"nicevalue\"}"
	response, err := worker.SendRequest(request)
	if err != nil {
		panic(err)
	}
	if response != request {
		panic("req != resp")
	}

	err = worker.SendSignal(syscall.SIGUSR2)
	if err != nil {
		panic(err)
	}
	worker.AwaitSignal(syscall.SIGUSR2)

	err = worker.TakeCheckpoint()
	if err != nil {
		panic(err)
	}

	// checkpoints := worker.GetCheckpoints()
	// stack1, err := checkpoints[0].GetMemory("[stack]")
	// if err != nil {
	// 	panic(err)
	// }
	// stack2, err := checkpoints[1].GetMemory("[stack]")
	// if err != nil {
	// 	panic(err)
	// }
	// start1, end1, process1 := stack1.GetOffsets()
	// start2, end2, process2 := stack2.GetOffsets()
	// fmt.Printf("start1: %d, end1: %d, process1: %d\n", start1, end1, process1)
	// fmt.Printf("start2: %d, end2: %d, process2: %d\n", start2, end2, process2)

	fmt.Print("> ")
	_, _ = buf.ReadBytes('\n')
	// time.Sleep(time.Second * 20)
	err = worker.Restore()
	if err != nil {
		panic(err)
	}

	fmt.Println("continued")
	function = "def handle(req):\n  print(req)\n  return {'new': True}"
	err = worker.SendFunction(function)
	if err != nil {
		panic(err)
	}
}

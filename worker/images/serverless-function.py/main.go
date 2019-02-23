package main

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
	"time"

	. "github.com/ostenbom/refunction/worker"
)

func main() {
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

	fmt.Printf("pid taken as %s, %d\n", pidString, wPid)

	err = worker.Activate()
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Millisecond * 100)

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
	// Expect(err).NotTo(HaveOccurred())
	// Expect(response).To(Equal(request))

	err = worker.SendSignal(syscall.SIGUSR2)
	if err != nil {
		panic(err)
	}
	err = worker.AwaitSignal(syscall.SIGUSR2)
	if err != nil {
		panic(err)
	}
	// time.Sleep(time.Second * 20)
	err = worker.Restore()
	if err != nil {
		panic(err)
	}

	fmt.Println("continued")
	go func() {
		pid, _ := worker.Pid()

		var waitStat syscall.WaitStatus
		for {
			_, err := syscall.Wait4(int(pid), &waitStat, 0, nil)
			if err != nil {
				fmt.Println("error with wait")
			}
			fmt.Printf("exited: %t\n", waitStat.Exited())
			fmt.Printf("exitstat: %d\n", waitStat.ExitStatus())
			fmt.Printf("signalled: %t\n", waitStat.Signaled())
			fmt.Printf("signal: %d\n", waitStat.Signal())
			fmt.Printf("stopped: %t\n", waitStat.Stopped())
			fmt.Printf("stopstat: %d\n", waitStat.StopSignal())

			err = syscall.PtraceCont(int(pid), int(waitStat.StopSignal()))
			if err != nil {
				fmt.Println("error with cont")
			}
			if waitStat.Exited() || waitStat.StopSignal() == 11 {
				return
			}
		}
	}()

	function = "def handle(req):\n  print(req)\n  return {'new': True}"
	err = worker.SendFunction(function)
	if err != nil {
		panic(err)
	}
}

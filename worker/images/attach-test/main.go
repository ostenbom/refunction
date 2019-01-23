package main

import (
	"fmt"
	"strconv"
	"syscall"
)

func main() {
	fmt.Println("what's the pid?")
	var input string
	fmt.Scanln(&input)

	pid, err := strconv.Atoi(input)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Going with %d.\n", pid)

	syscall.PtraceAttach(pid)

	var wstat syscall.WaitStatus
	thing, err := syscall.Wait4(pid, &wstat, 0, nil)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Returned f. wait: %d", thing)

	fmt.Println("Attached")
	fmt.Scanln(&input)

	syscall.PtraceCont(pid, 0)

	fmt.Println("Cont.. ")
	fmt.Scanln(&input)

	err = syscall.Kill(pid, syscall.SIGUSR1)
	if err != nil {
		panic(err)
	}

	_, err = syscall.Wait4(pid, &wstat, 0, nil)
	if err != nil {
		panic(err)
	}

	sig := wstat.StopSignal()

	fmt.Printf("stopped: %t, signal: %d", wstat.Stopped(), sig)

	fmt.Println("Woke with SIGUSR1")
	fmt.Scanln(&input)

	syscall.PtraceCont(pid, 10)

	// syscall.PtraceCont(pid, 0)
	//
	fmt.Println("continued")
	fmt.Scanln(&input)

	// fmt.Println("sent 'go'")
	// fmt.Scanln(&input)
	//
	// syscall.PtraceCont(pid, 0)
	//
	// fmt.Println("continued")
	// fmt.Scanln(&input)
	//
	err = syscall.Kill(pid, syscall.SIGSTOP)
	if err != nil {
		panic(err)
	}

	fmt.Println("stopped")
	fmt.Scanln(&input)

	fmt.Println("to detach")
	fmt.Scanln(&input)

	syscall.PtraceDetach(pid)

	fmt.Println("detached")
	fmt.Scanln(&input)
}

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := false

	go func() {
		<-sigs
		done = true
	}()

	_, _, err := syscall.Syscall(syscall.SYS_PTRACE, syscall.PTRACE_TRACEME, 0, 0)
	if err != 0 {
		panic(err)
	}

	for !done {
		fmt.Println("sleeping for 5")
		time.Sleep(time.Second * 5)
	}
}

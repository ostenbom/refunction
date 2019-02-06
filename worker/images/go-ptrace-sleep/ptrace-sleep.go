package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {
	// Allow graceful stopping
	stopSigs := make(chan os.Signal, 1)
	signal.Notify(stopSigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-stopSigs
		os.Exit(0)
	}()

	// Signal to exit busy loop
	readySig := make(chan os.Signal, 1)
	signal.Notify(readySig, syscall.SIGUSR1)

	ready := false
	go func() {
		<-readySig
		ready = true
	}()

	_, _, err := syscall.Syscall(syscall.SYS_PTRACE, syscall.PTRACE_TRACEME, 0, 0)
	if err != 0 {
		panic(err)
	}

	for !ready {
		time.Sleep(10)
	}

	f, openErr := os.OpenFile("/tmp/count.txt", os.O_CREATE|os.O_TRUNC|os.O_APPEND|os.O_WRONLY, 0600)
	if openErr != nil {
		panic(openErr)
	}
	defer f.Close()

	count := 0
	for true {
		fmt.Println("sleeping for 2")
		time.Sleep(time.Second * 2)

		_, err := f.WriteString(strconv.Itoa(count))
		if err != nil {
			panic(err)
		}

		count++
	}
}

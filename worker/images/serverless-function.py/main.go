package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"syscall"
	"text/tabwriter"

	. "github.com/ostenbom/refunction/worker"
	sec "github.com/seccomp/libseccomp-golang"
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

	go func() {
		var regs syscall.PtraceRegs
		var ss syscallCounter
		var waitStat syscall.WaitStatus
		ss = ss.init()
		exit := true

		_, err = syscall.Wait4(pid, &waitStat, 0, nil)
		if err != nil {
			fmt.Println("error with wait")
		}

		for {
			if exit {
				fmt.Println("waiting for regs")
				err = syscall.PtraceGetRegs(pid, &regs)
				if err != nil {
					fmt.Printf("oh err: %s", err)
					err = syscall.PtraceSyscall(pid, 0)
					if err != nil {
						panic(err)
					}
					continue
				}

				fmt.Println("name")
				// Uncomment to show each syscall as it's called
				name := ss.getName(regs.Orig_rax)
				fmt.Printf("%s\n", name)
				ss.inc(regs.Orig_rax)
			}

			fmt.Println("doing syscall")
			err = syscall.PtraceSyscall(pid, 0)
			if err != nil {
				panic(err)
			}

			_, err = syscall.Wait4(pid, &waitStat, 0, nil)
			if err != nil {
				fmt.Println("error with wait")
			}
			fmt.Printf("exited: %t\n", waitStat.Exited())
			fmt.Printf("exitstat: %d\n", waitStat.ExitStatus())
			fmt.Printf("signalled: %t\n", waitStat.Signaled())
			fmt.Printf("signal: %d\n", waitStat.Signal())
			fmt.Printf("stopped: %t\n", waitStat.Stopped())
			fmt.Printf("stopstat: %d\n", waitStat.StopSignal())

			exit = !exit
		}

		ss.print()
	}()

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
	// go func() {
	// 	pid, _ := worker.Pid()
	//
	// 	var waitStat syscall.WaitStatus
	// 	for {
	// 		_, err := syscall.Wait4(int(pid), &waitStat, 0, nil)
	// 		if err != nil {
	// 			fmt.Println("error with wait")
	// 		}
	// 		fmt.Printf("exited: %t\n", waitStat.Exited())
	// 		fmt.Printf("exitstat: %d\n", waitStat.ExitStatus())
	// 		fmt.Printf("signalled: %t\n", waitStat.Signaled())
	// 		fmt.Printf("signal: %d\n", waitStat.Signal())
	// 		fmt.Printf("stopped: %t\n", waitStat.Stopped())
	// 		fmt.Printf("stopstat: %d\n", waitStat.StopSignal())
	//
	// 		err = syscall.PtraceCont(int(pid), int(waitStat.StopSignal()))
	// 		if err != nil {
	// 			fmt.Println("error with cont")
	// 		}
	// 		if waitStat.Exited() || waitStat.StopSignal() == 11 {
	// 			return
	// 		}
	// 	}
	// }()
	//
	function = "def handle(req):\n  print(req)\n  return {'new': True}"
	err = worker.SendFunction(function)
	if err != nil {
		panic(err)
	}
}

type syscallCounter []int

const maxSyscalls = 303

func (s syscallCounter) init() syscallCounter {
	s = make(syscallCounter, maxSyscalls)
	return s
}

func (s syscallCounter) inc(syscallID uint64) error {
	if syscallID > maxSyscalls {
		return fmt.Errorf("invalid syscall ID (%x)", syscallID)
	}

	s[syscallID]++
	return nil
}

func (s syscallCounter) print() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 8, ' ', tabwriter.AlignRight|tabwriter.Debug)
	for k, v := range s {
		if v > 0 {
			name, _ := sec.ScmpSyscall(k).GetName()
			fmt.Fprintf(w, "%d\t%s\n", v, name)
		}
	}
	w.Flush()
}

func (s syscallCounter) getName(syscallID uint64) string {
	name, _ := sec.ScmpSyscall(syscallID).GetName()
	return name
}

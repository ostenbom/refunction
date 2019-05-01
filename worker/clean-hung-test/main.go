package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	runDir := os.Args[1]
	fmt.Println(runDir)
	if len(os.Args) > 2 {
		pid := os.Args[2]
		fmt.Println("killing pid")
		err := exec.Command("kill", "-9", string(pid)).Run()
		if err != nil {
			panic(err)
		}
	}

	out, _ := exec.Command("grep", runDir, "/proc/mounts").Output()
	if string(out) != "" {
		fmt.Println("doing unmount")
		mount := strings.Fields(string(out))[1]
		_, err := exec.Command("umount", "-r", mount).Output()
		if err != nil {
			panic(err)
		}
	}

	os.RemoveAll(runDir)
	// Ignore errors
	os.RemoveAll("/var/run/containerd/runc/refunction-worker0")
	os.RemoveAll("/var/run/containerd/runc/refunction-worker1")
	os.RemoveAll("/var/run/containerd/runc/refunction-worker2")
	os.RemoveAll("/var/run/containerd/runc/refunction-worker3")
	os.RemoveAll("/var/run/containerd/runc/refunction-worker4")
}

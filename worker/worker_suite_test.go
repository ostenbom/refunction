package worker_test

import (
	"os/exec"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestWorker(t *testing.T) {
	containerd, err := StartContainerd()
	if err != nil {
		Fail("Failed to start containerd server")
	}

	RegisterFailHandler(Fail)
	RunSpecs(t, "Worker Suite")

	err = containerd.Process.Kill()
	if err != nil {
		Fail("Could not kill containerd server")
	}
}

func StartContainerd() (*exec.Cmd, error) {
	cmd := exec.Command("containerd")
	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	return cmd, nil
}

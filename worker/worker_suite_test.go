package worker_test

import (
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestWorker(t *testing.T) {
	containerd, err := StartContainerd()
	if err != nil {
		panic("Failed to start containerd server")
	}

	RegisterFailHandler(Fail)
	RunSpecs(t, "Worker Suite")

	err = containerd.Process.Signal(syscall.SIGINT)
	if err != nil {
		panic("Could not kill containerd server")
	}

	_, err = containerd.Process.Wait()
	if err != nil {
		panic("Containerd server did not stop")
	}
}

func StartContainerd() (*exec.Cmd, error) {
	configPath, err := filepath.Abs("config.toml")
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("containerd", "-c", configPath)
	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	return cmd, nil
}

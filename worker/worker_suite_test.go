package worker_test

import (
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/containerd/containerd"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestWorker(t *testing.T) {
	var createContainerd bool
	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		createContainerd = true
	} else {
		client.Close()
		createContainerd = false
	}

	var server *exec.Cmd
	if createContainerd {
		server, err = StartContainerd()
		if err != nil {
			panic("Failed to start containerd server")
		}
	}

	RegisterFailHandler(Fail)
	RunSpecs(t, "Worker Suite")

	if createContainerd {
		err = server.Process.Signal(syscall.SIGINT)
		if err != nil {
			panic("Could not kill containerd server")
		}

		_, err = server.Process.Wait()
		if err != nil {
			panic("Containerd server did not stop")
		}
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

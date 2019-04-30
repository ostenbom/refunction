package workerpool

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/burntsushi/toml"
	"github.com/containerd/containerd"
	"github.com/ostenbom/refunction/worker"
	"github.com/ostenbom/refunction/worker/containerdrunner"
)

const defaultRuntime = "alpinepython"
const defaultTarget = "embedded-python"

type WorkerPool struct {
	workers []*worker.Worker
	server  *exec.Cmd
	client  *containerd.Client
	config  containerdrunner.Config
	runDir  string
}

func NewWorkerPool(size int) (*WorkerPool, error) {
	runDir, err := ioutil.TempDir("", "refunction")
	if err != nil {
		return nil, fmt.Errorf("could not create temp dir for worker pool: %s", err)
	}

	config := containerdrunner.ContainerdConfig(runDir)
	server, err := NewContainerdServer(runDir, config)
	if err != nil {
		return nil, fmt.Errorf("could not start contianerd server: %s", err)
	}

	client, err := containerd.New(config.GRPC.Address)
	if err != nil {
		return nil, fmt.Errorf("could not connect to containerd client: %s", err)
	}

	workers := make([]*worker.Worker, size)
	for i := 0; i < size; i++ {
		w, err := worker.NewWorker(strconv.Itoa(i), client, defaultRuntime, defaultTarget)
		if err != nil {
			return nil, fmt.Errorf("could not start worker in pool: %s", err)
		}
		workers[i] = w
	}

	return &WorkerPool{
		workers: workers,
		server:  server,
		client:  client,
		config:  config,
		runDir:  runDir,
	}, nil
}

func NewContainerdServer(runDir string, config containerdrunner.Config) (*exec.Cmd, error) {

	configFile, err := os.OpenFile(filepath.Join(runDir, "containerd.toml"), os.O_TRUNC|os.O_WRONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("could not open config file: %s", err)
	}
	err = toml.NewEncoder(configFile).Encode(&config)
	if err != nil {
		return nil, fmt.Errorf("could not encode config: %s", err)
	}

	err = configFile.Close()
	if err != nil {
		return nil, fmt.Errorf("could not close config file: %s", err)
	}

	cmd := exec.Command("containerd", "--config", configFile.Name())
	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("could exec containerd: %s", err)
	}

	return cmd, nil
}

func (p *WorkerPool) Close() error {
	var workerErr error
	for _, w := range p.workers {
		err := w.End()
		if err != nil {
			workerErr = err
		}
	}
	if workerErr != nil {
		return workerErr
	}

	err := p.client.Close()
	if err != nil {
		return err
	}

	err = p.server.Process.Kill()
	if err != nil {
		return err
	}

	err = os.RemoveAll(p.runDir)
	if err != nil {
		return err
	}

	return nil
}

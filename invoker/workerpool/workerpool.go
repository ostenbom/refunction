package workerpool

import (
	"archive/tar"
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/burntsushi/toml"
	"github.com/containerd/containerd"
	"github.com/ostenbom/refunction/worker"
	"github.com/ostenbom/refunction/worker/containerdrunner"
)

const defaultRuntime = "alpinepython"
const defaultTarget = "embedded-python"

type WorkerPool struct {
	workers   []*worker.Worker
	server    *exec.Cmd
	client    *containerd.Client
	config    containerdrunner.Config
	runDir    string
	scheduler *Scheduler
}

func NewWorkerPool(size int) (*WorkerPool, error) {
	runDir, err := ioutil.TempDir("", "refunction")
	if err != nil {
		return nil, fmt.Errorf("could not create temp dir for worker pool: %s", err)
	}

	cacheDir := "/var/cache/refunction"
	err = ensureRuntimes([]string{defaultRuntime}, cacheDir)
	if err != nil {
		return nil, err
	}

	err = ensureLayers([]string{defaultTarget}, cacheDir)
	if err != nil {
		return nil, err
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

	for _, w := range workers {
		err := w.Start()
		if err != nil {
			return nil, fmt.Errorf("could not start worker: %s", err)
		}
		err = w.Activate()
		if err != nil {
			return nil, fmt.Errorf("could not activate worker: %s", err)
		}
	}

	scheduler := NewScheduler(workers)

	return &WorkerPool{
		workers:   workers,
		server:    server,
		client:    client,
		config:    config,
		runDir:    runDir,
		scheduler: scheduler,
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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
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
	p.server.Wait()

	err = os.RemoveAll(p.runDir)
	if err != nil {
		return err
	}

	return nil
}

func ensureRuntimes(runtimes []string, workDir string) error {
	err := os.MkdirAll(fmt.Sprintf("%s/runtimes", workDir), os.ModePerm)
	if err != nil {
		return fmt.Errorf("could not create runtime cache dir")
	}

	for _, runtime := range runtimes {
		err := downloadRuntime(runtime, workDir)
		if err != nil {
			return fmt.Errorf("could not download %s: %s", runtime, err)
		}
	}

	return nil
}

func ensureLayers(layers []string, workDir string) error {
	layersPath := fmt.Sprintf("%s/activelayers", workDir)
	err := os.MkdirAll(layersPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("could not create layers cache dir")
	}

	for _, layer := range layers {
		err := downloadLayer(layer, workDir)
		if err != nil {
			return fmt.Errorf("could not download %s: %s", layer, err)
		}
	}

	return nil
}

func downloadRuntime(runtime, workDir string) error {
	runtimePath := fmt.Sprintf("%s/runtimes/%s", workDir, runtime)
	runtimeURL := fmt.Sprintf("https://s3.eu-west-2.amazonaws.com/refunction-runtimes/%s.tar", runtime)

	err := os.MkdirAll(runtimePath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("could not make runtimePath %s: %s", runtime, err)
	}

	resp, err := http.Get(runtimeURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	firstSection, err := reader.Peek(150)
	if err != nil {
		return fmt.Errorf("received runtime too small to peek: %s", err)
	}
	if strings.Contains(string(firstSection), "AccessDenied") {
		return fmt.Errorf("runtime %s could not be donwnloaded/did not exist", runtime)
	}

	err = untar(reader, runtimePath)
	if err != nil {
		return fmt.Errorf("could not untar runtime %s: %s", runtime, err)
	}

	return nil
}

func downloadLayer(layerName, workDir string) error {
	layerDir := fmt.Sprintf("%s/activelayers/%s", workDir, layerName)
	err := os.MkdirAll(layerDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("could not create layer %s dir: %s", layerName, err)
	}

	layerURL := fmt.Sprintf("https://s3.eu-west-2.amazonaws.com/refunction-runtimes/layers/%s/layer.tar", layerName)
	resp, err := http.Get(layerURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(fmt.Sprintf("%s/layer.tar", layerDir))
	if err != nil {
		return err
	}
	defer out.Close()

	reader := bufio.NewReader(resp.Body)
	firstSection, err := reader.Peek(150)
	if err != nil {
		return fmt.Errorf("received layer too small to peek: %s", err)
	}
	if strings.Contains(string(firstSection), "AccessDenied") {
		return fmt.Errorf("layer %s could not be donwnloaded/did not exist", layerName)
	}

	_, err = io.Copy(out, reader)
	return err
}

// https://medium.com/@skdomino/taring-untaring-files-in-go-6b07cf56bc07
func untar(r io.Reader, path string) error {
	tr := tar.NewReader(r)

	for {
		header, err := tr.Next()

		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		case header == nil:
			continue
		}

		target := filepath.Join(path, header.Name)

		switch header.Typeflag {

		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
					return err
				}
			}

		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			f.Close()
		}
	}
}

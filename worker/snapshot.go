package worker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/archive"
	"github.com/containerd/containerd/mount"
	"github.com/containerd/containerd/snapshots"
)

func NewSnapshotManager(ctx context.Context, client *containerd.Client, runtime string) (*SnapshotManager, error) {
	cacheDir := "/var/cache/refunction"
	// If running tests..
	workDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	if strings.Contains(workDir, "refunction/worker") {
		cacheDir = workDir
	}

	opt := snapshots.WithLabels(map[string]string{
		"containerd.io/gc.root": time.Now().UTC().Format(time.RFC3339),
	})

	layersPath := fmt.Sprintf("%s/activelayers", cacheDir)
	manager := SnapshotManager{
		runtime:     runtime,
		ctx:         ctx,
		layersPath:  layersPath,
		snapshotter: client.SnapshotService("overlayfs"),
		noGc:        opt,
	}

	err = manager.ensureRuntimeBase(cacheDir)
	if err != nil {
		return nil, fmt.Errorf("could not create runtime base: %s", err)
	}

	return &manager, nil
}

type SnapshotManager struct {
	runtime     string
	ctx         context.Context
	layersPath  string
	snapshotter snapshots.Snapshotter
	noGc        snapshots.Opt
}

type manifest struct {
	Layers []string
}

func (m *SnapshotManager) ensureRuntimeBase(workDir string) error {
	runtimeManifestBytes, err := ioutil.ReadFile(fmt.Sprintf("%s/runtimes/%s/manifest.json", workDir, m.runtime))
	if err != nil {
		return fmt.Errorf("could not open runtime dir: %s", err)
	}

	var runmen []manifest
	err = json.Unmarshal(runtimeManifestBytes, &runmen)
	if err != nil {
		return fmt.Errorf("could not parse layers from manifest: %s", err)
	}
	if len(runmen) == 0 {
		return fmt.Errorf("no images in runtime manifest")
	}
	runman := runmen[0]

	// Sanity check
	if len(runman.Layers) == 0 {
		return fmt.Errorf("no layers in runtime manifest")
	}

	prevLayer := ""
	for i := 0; i < len(runman.Layers); i++ {
		currentLayer := runman.Layers[i]
		layerPath := fmt.Sprintf("%s/runtimes/%s/%s", workDir, m.runtime, currentLayer)

		layerName := currentLayer
		if i == len(runman.Layers)-1 {
			layerName = m.runtime
		}
		err := m.createLayer(layerName, layerPath, prevLayer)
		if err != nil {
			return fmt.Errorf("could not create runtime layer: %s", err)
		}
		prevLayer = currentLayer
	}

	return nil
}

func (m *SnapshotManager) CreateLayerFromBase(layerName string) error {
	layerPath := fmt.Sprintf("%s/%s/layer.tar", m.layersPath, layerName)
	return m.createLayer(layerName, layerPath, m.runtime)
}

func (m *SnapshotManager) CreateRoView(layerName, containerName string) ([]mount.Mount, error) {
	mounts, err := m.snapshotter.View(m.ctx, containerName, layerName, m.noGc)
	if err != nil {
		return nil, err
	}

	return mounts, nil
}

func (m *SnapshotManager) GetRwMounts(layerName, containerName string) ([]mount.Mount, error) {
	mounts, err := m.snapshotter.Prepare(m.ctx, containerName, layerName, m.noGc)
	if err != nil {
		return nil, err
	}

	return mounts, nil
}

func (m *SnapshotManager) createLayer(layerName, layerPath, parentName string) error {
	_, err := m.snapshotter.Stat(m.ctx, layerName)
	if err == nil {
		return nil
	}

	tmpDir, err := ioutil.TempDir("", "snapshotmanager")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	activeLayerName := fmt.Sprintf("%s-active", layerName)
	mounts, err := m.snapshotter.Prepare(m.ctx, activeLayerName, parentName, m.noGc)
	if err != nil {
		return err
	}

	// Mount it to the tempdir
	err = mount.All(mounts, tmpDir)
	if err != nil {
		return fmt.Errorf("all mount error: %s", err)
	}
	defer mount.UnmountAll(tmpDir, 0)

	layerTar, err := os.Open(layerPath)
	if err != nil {
		return err
	}
	r := bufio.NewReader(layerTar)

	_, err = archive.Apply(m.ctx, tmpDir, r)
	if err != nil {
		return fmt.Errorf("could not apply error: %s", err)
	}

	// Read any trailing data
	_, err = io.Copy(ioutil.Discard, r)
	if err != nil {
		return fmt.Errorf("could not read trailing data: %s", err)
	}

	err = m.snapshotter.Commit(m.ctx, layerName, activeLayerName, m.noGc)
	if err != nil {
		return fmt.Errorf("commit error: %s", err)
	}

	return nil
}

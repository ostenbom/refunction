package worker

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/archive"
	"github.com/containerd/containerd/mount"
	"github.com/containerd/containerd/snapshots"
)

func NewSnapshotManager(client *containerd.Client, ctx context.Context) (*SnapshotManager, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("could not get working directory: %s", err)
	}

	opt := snapshots.WithLabels(map[string]string{
		"containerd.io/gc.root": time.Now().UTC().Format(time.RFC3339),
	})

	manager := SnapshotManager{
		baseName:    "alpine",
		ctx:         ctx,
		layersPath:  fmt.Sprintf("%s/layers", workDir),
		snapshotter: client.SnapshotService("overlayfs"),
		noGc:        opt,
	}

	err = manager.EnsureBaseLayer()
	if err != nil {
		return nil, fmt.Errorf("could not create base layer: %s", err)
	}

	return &manager, nil
}

type SnapshotManager struct {
	baseName    string
	ctx         context.Context
	layersPath  string
	snapshotter snapshots.Snapshotter
	noGc        snapshots.Opt
}

func (m *SnapshotManager) EnsureBaseLayer() error {
	return m.createLayer(m.baseName, "")
}

func (m *SnapshotManager) CreateLayerFromBase(layerName string) error {
	return m.createLayer(layerName, m.baseName)
}

func (m *SnapshotManager) GetRwMounts(layerName, containerName string) ([]mount.Mount, error) {
	mounts, err := m.snapshotter.Prepare(m.ctx, containerName, layerName, m.noGc)
	if err != nil {
		return nil, err
	}

	return mounts, nil
}

func (m *SnapshotManager) createLayer(layerName, parentName string) error {
	_, err := m.snapshotter.Stat(m.ctx, layerName)
	if err == nil {
		return nil
	}

	layerPath := fmt.Sprintf("%s/%s/layer.tar", m.layersPath, layerName)
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

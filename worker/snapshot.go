package worker

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/mount"
	"github.com/containerd/containerd/snapshots"
	"github.com/openSUSE/umoci/oci/layer"
)

func NewSnapshotManager(client *containerd.Client, ctx context.Context) (*SnapshotManager, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("could not get working directory: %s", err)
	}

	manager := SnapshotManager{
		baseName:    "alpine",
		ctx:         ctx,
		layersPath:  fmt.Sprintf("%s/layers", workDir),
		snapshotter: client.SnapshotService("overlayfs"),
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
}

func (m *SnapshotManager) EnsureBaseLayer() error {
	baseLayerPath := fmt.Sprintf("%s/alpine/layer.tar", m.layersPath)
	tmpDir, err := ioutil.TempDir("", "snapshotmanager")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// Start an empty base layer
	emptyBaseKey := "emptybase"
	mounts, err := m.snapshotter.Prepare(m.ctx, emptyBaseKey, "")
	if err != nil {
		return err
	}
	defer m.snapshotter.Remove(m.ctx, emptyBaseKey)

	// Mount it to the tempdir
	if err := mount.All(mounts, tmpDir); err != nil {
		return err
	}
	defer mount.UnmountAll(tmpDir, 0)

	layerTar, err := os.Open(baseLayerPath)
	if err != nil {
		return err
	}

	err = layer.UnpackLayer(tmpDir, layerTar, nil)
	if err != nil {
		return err
	}
	time.Sleep(time.Minute)

	m.snapshotter.Commit(m.ctx, "alpine", emptyBaseKey)

	return nil
}

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

	var opt = snapshots.WithLabels(map[string]string{
		"containerd.io/gc.root": time.Now().UTC().Format(time.RFC3339),
	})
	// Start an empty base layer
	emptyBaseKey := "emptybase"
	mounts, err := m.snapshotter.Prepare(m.ctx, emptyBaseKey, "", opt)
	if err != nil {
		return err
	}

	// Mount it to the tempdir
	if err := mount.All(mounts, tmpDir); err != nil {
		return fmt.Errorf("all mount error: %s", err)
	}
	defer mount.UnmountAll(tmpDir, 0)

	layerTar, err := os.Open(baseLayerPath)
	if err != nil {
		return err
	}
	r := bufio.NewReader(layerTar)

	_, err = archive.Apply(m.ctx, tmpDir, r)
	if err != nil {
		return fmt.Errorf("could not apply error: %s", err)
	}

	// Read any trailing data
	if _, err := io.Copy(ioutil.Discard, r); err != nil {
		return fmt.Errorf("could not read trailing data: %s", err)
	}

	err = m.snapshotter.Commit(m.ctx, "alpine", emptyBaseKey, opt)
	if err != nil {
		return fmt.Errorf("commit error: %s", err)
	}

	return nil
}

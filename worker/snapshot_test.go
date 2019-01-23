package worker_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/snapshots"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/ostenbom/refunction/worker"
)

var _ = Describe("Snapshot manager", func() {
	var snapshotter snapshots.Snapshotter
	var ctx context.Context
	baseName := "alpine"

	BeforeEach(func() {
		snapshotter = client.SnapshotService("overlayfs")
		id := strconv.Itoa(GinkgoParallelNode())
		ctx = namespaces.WithNamespace(context.Background(), "refunction-worker"+id)
	})

	Context("when there was no base layer", func() {
		BeforeEach(func() {
			snapshotter.Remove(ctx, baseName)
		})

		It("creates a base layer on creating the manager", func() {

			entriesBefore, err := getSnapshotEntries()
			Expect(err).NotTo(HaveOccurred())

			_, err = NewSnapshotManager(client, ctx)
			Expect(err).NotTo(HaveOccurred())

			info, err := snapshotter.Stat(ctx, baseName)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Name).To(Equal(baseName))

			entriesAfter, err := getSnapshotEntries()
			Expect(err).NotTo(HaveOccurred())

			Expect(len(entriesAfter)).To(Equal(len(entriesBefore) + 1))
		})
	})

	Context("when there was a base layer", func() {
		It("doesn't create any new layers", func() {
			Expect(makeBaseLayer(ctx, snapshotter, baseName)).To(Succeed())

			entriesBefore, err := getSnapshotEntries()
			Expect(err).NotTo(HaveOccurred())

			// first base name exists
			info, err := snapshotter.Stat(ctx, baseName)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Name).To(Equal(baseName))

			_, err = NewSnapshotManager(client, ctx)
			Expect(err).NotTo(HaveOccurred())

			// then nothing new was made
			entriesAfter, err := getSnapshotEntries()
			Expect(err).NotTo(HaveOccurred())

			Expect(len(entriesAfter)).To(Equal(len(entriesBefore)))
		})
	})

	Context("when the manager has been created", func() {
		var manager *SnapshotManager
		var startEntries []os.FileInfo
		BeforeEach(func() {
			var err error
			manager, err = NewSnapshotManager(client, ctx)
			Expect(err).NotTo(HaveOccurred())

			startEntries, err = getSnapshotEntries()
			Expect(err).NotTo(HaveOccurred())
		})

		It("can create a layer on top of the base", func() {
			layerName := "echo-hello"
			err := manager.CreateLayerFromBase(layerName)
			Expect(err).NotTo(HaveOccurred())

			info, err := snapshotter.Stat(ctx, layerName)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Name).To(Equal(layerName))
			Expect(info.Parent).To(Equal(baseName))

			entriesAfter, err := getSnapshotEntries()
			Expect(err).NotTo(HaveOccurred())

			Expect(len(entriesAfter)).To(Equal(len(startEntries) + 1))
		})

		It("can get rw mounts of a layer", func() {
			mounts, err := manager.GetRwMounts("alpine", "mycontainer")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(mounts)).NotTo(Equal(0))
		})
	})

})

func getSnapshotEntries() ([]os.FileInfo, error) {
	snapshotsDir := fmt.Sprintf("%s/%s", config.Root, "io.containerd.snapshotter.v1.overlayfs/snapshots")
	entries, err := ioutil.ReadDir(snapshotsDir)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

func makeBaseLayer(ctx context.Context, snapshotter snapshots.Snapshotter, baseName string) error {
	opt := snapshots.WithLabels(map[string]string{
		"containerd.io/gc.root": time.Now().UTC().Format(time.RFC3339),
	})
	_, err := snapshotter.Prepare(ctx, "emptybase", "", opt)
	if err != nil {
		return err
	}

	err = snapshotter.Commit(ctx, baseName, "emptybase", opt)
	if err != nil {
		return err
	}
	return nil
}

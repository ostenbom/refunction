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
	var runtime string

	BeforeEach(func() {
		runtime = "alpine"
		snapshotter = client.SnapshotService("overlayfs")
		id := strconv.Itoa(GinkgoParallelNode())
		ctx = namespaces.WithNamespace(context.Background(), "refunction-worker"+id)
	})

	Context("when there was no prepared runtime", func() {
		BeforeEach(func() {
			snapshotter.Remove(ctx, runtime)
		})

		It("commits a runtime snapshot on creating the manager", func() {

			entriesBefore, err := getSnapshotEntries()
			Expect(err).NotTo(HaveOccurred())

			_, err = NewSnapshotManager(ctx, client, runtime)
			Expect(err).NotTo(HaveOccurred())

			info, err := snapshotter.Stat(ctx, runtime)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Name).To(Equal(runtime))

			entriesAfter, err := getSnapshotEntries()
			Expect(err).NotTo(HaveOccurred())

			Expect(len(entriesAfter)).To(Equal(len(entriesBefore) + 1))
		})
	})

	Context("when there was an existing runtime", func() {
		It("doesn't create any new layers", func() {
			Expect(makeBaseLayer(ctx, snapshotter, runtime)).To(Succeed())

			entriesBefore, err := getSnapshotEntries()
			Expect(err).NotTo(HaveOccurred())

			// first base name exists
			info, err := snapshotter.Stat(ctx, runtime)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Name).To(Equal(runtime))

			_, err = NewSnapshotManager(ctx, client, runtime)
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
			manager, err = NewSnapshotManager(ctx, client, runtime)
			Expect(err).NotTo(HaveOccurred())

			startEntries, err = getSnapshotEntries()
			Expect(err).NotTo(HaveOccurred())
		})

		It("can create a layer on top of the runtime", func() {
			layerName := "echo-hello"
			err := manager.CreateLayerFromBase(layerName)
			Expect(err).NotTo(HaveOccurred())

			info, err := snapshotter.Stat(ctx, layerName)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Name).To(Equal(layerName))
			Expect(info.Parent).To(Equal(runtime))

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

	Context("when runtime has multiple layers", func() {
		BeforeEach(func() {
			runtime = "python"
		})

		It("creates all runtime layers", func() {
			workDir, err := os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			runtimeDir, err := os.Open(fmt.Sprintf("%s/runtimes/%s", workDir, runtime))
			Expect(err).NotTo(HaveOccurred())
			runtimeLayers, err := runtimeDir.Readdirnames(0)
			Expect(err).NotTo(HaveOccurred())

			_, err = NewSnapshotManager(ctx, client, runtime)
			Expect(err).NotTo(HaveOccurred())

			snapshotsAfter, err := getSnapshotEntries()
			Expect(err).NotTo(HaveOccurred())

			// -3 for manifest, repositories & config.json
			Expect(len(runtimeLayers) - 3).To(Equal(len(snapshotsAfter)))
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

func makeBaseLayer(ctx context.Context, snapshotter snapshots.Snapshotter, runtime string) error {
	opt := snapshots.WithLabels(map[string]string{
		"containerd.io/gc.root": time.Now().UTC().Format(time.RFC3339),
	})
	_, err := snapshotter.Prepare(ctx, "emptybase", "", opt)
	if err != nil {
		return err
	}

	err = snapshotter.Commit(ctx, runtime, "emptybase", opt)
	if err != nil {
		return err
	}
	return nil
}

package worker_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/snapshots"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/ostenbom/refunction/worker"
)

var _ = Describe("Fs", func() {
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

})

func getSnapshotEntries() ([]os.FileInfo, error) {
	snapshotsDir := fmt.Sprintf("%s/%s", config.Root, "io.containerd.snapshotter.v1.overlayfs/snapshots")
	entries, err := ioutil.ReadDir(snapshotsDir)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

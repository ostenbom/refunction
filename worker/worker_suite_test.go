package worker_test

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"

	"code.cloudfoundry.org/guardian/gqt/containerdrunner"
	"github.com/burntsushi/toml"
	"github.com/containerd/containerd"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	runDir string
	client *containerd.Client
	server *exec.Cmd
)

func TestWorker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Worker Suite")
}

var _ = BeforeEach(func() {
	runDir, err := ioutil.TempDir("", "")
	Expect(err).NotTo(HaveOccurred())

	config := containerdrunner.ContainerdConfig(runDir)
	server = NewServer(runDir, config)

	client, err = GetContainerdClient(config.GRPC.Address)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterEach(func() {
	err := client.Close()
	Expect(err).NotTo(HaveOccurred())

	err = server.Process.Signal(syscall.SIGINT)
	Expect(err).NotTo(HaveOccurred())
	_, err = server.Process.Wait()
	Expect(err).NotTo(HaveOccurred())

	Expect(os.RemoveAll(runDir)).To(Succeed())

})

func NewServer(runDir string, config containerdrunner.Config) *exec.Cmd {
	configFile, err := os.OpenFile(filepath.Join(runDir, "containerd.toml"), os.O_TRUNC|os.O_WRONLY|os.O_CREATE, os.ModePerm)
	Expect(err).NotTo(HaveOccurred())
	Expect(toml.NewEncoder(configFile).Encode(&config)).To(Succeed())
	Expect(configFile.Close()).To(Succeed())

	cmd := exec.Command("containerd", "--config", configFile.Name())
	err = cmd.Start()
	Expect(err).NotTo(HaveOccurred())
	return cmd
}

func GetContainerdClient(socketAddr string) (*containerd.Client, error) {
	return containerd.New(socketAddr)
}

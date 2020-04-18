package worker_test

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/containerd/containerd"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/ostenbom/refunction/worker/containerdrunner"
)

var (
	runDir string
	config containerdrunner.Config
	client *containerd.Client
	server *gexec.Session
)

func TestWorker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Worker Suite")
}

var _ = BeforeEach(func() {
	var err error
	runDir, err = ioutil.TempDir("", "")
	Expect(err).NotTo(HaveOccurred())

	config = containerdrunner.ContainerdConfig(runDir)
	server = NewServer(runDir, config)

	client, err = GetContainerdClient(config.GRPC.Address)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterEach(func() {
	err := client.Close()
	Expect(err).NotTo(HaveOccurred())

	Expect(server.Terminate().Wait()).To(gexec.Exit(0))

	Expect(os.RemoveAll(runDir)).To(Succeed())
})

func NewServer(runDir string, config containerdrunner.Config) *gexec.Session {
	configFile, err := os.OpenFile(filepath.Join(runDir, "containerd.toml"), os.O_TRUNC|os.O_WRONLY|os.O_CREATE, os.ModePerm)
	Expect(err).NotTo(HaveOccurred())
	Expect(toml.NewEncoder(configFile).Encode(&config)).To(Succeed())
	Expect(configFile.Close()).To(Succeed())

	cmd := exec.Command("containerd", "--config", configFile.Name())
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	return session
}

func GetContainerdClient(socketAddr string) (*containerd.Client, error) {
	return containerd.New(socketAddr)
}

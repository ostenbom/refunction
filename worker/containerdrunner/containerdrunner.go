package containerdrunner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

type Config struct {
	Root            string      `toml:"root"`
	State           string      `toml:"state"`
	Subreaper       bool        `toml:"subreaper"`
	OomScore        int         `toml:"oom_score"`
	GRPC            GRPCConfig  `toml:"grpc"`
	Debug           DebugConfig `toml:"debug"`
	DisabledPlugins []string    `toml:"disabled_plugins"`
	Plugins         Plugins     `toml:"plugins"`

	RunDir string
}

type GRPCConfig struct {
	Address string `toml:"address"`
}

type DebugConfig struct {
	Address string `toml:"address"`
	Level   string `toml:"level"`
}

type Plugins struct {
	Linux Linux `toml:"linux"`
}

type Linux struct {
	ShimDebug bool `toml:"shim_debug"`
}

func ContainerdConfig(containerdDataDir string) Config {
	return Config{
		Root:      filepath.Join(containerdDataDir, "root"),
		State:     filepath.Join(containerdDataDir, "state"),
		Subreaper: true,
		OomScore:  -999,
		GRPC: GRPCConfig{
			Address: filepath.Join(containerdDataDir, "containerd.sock"),
		},
		Debug: DebugConfig{
			Address: filepath.Join(containerdDataDir, "debug.sock"),
			Level:   "debug",
		},
		DisabledPlugins: []string{
			"cri",
		},
		Plugins: Plugins{
			Linux: Linux{
				ShimDebug: true,
			},
		},
	}
}

func NewSession(runDir string, config Config) *gexec.Session {
	configFile, err := os.OpenFile(filepath.Join(runDir, "containerd.toml"), os.O_TRUNC|os.O_WRONLY|os.O_CREATE, os.ModePerm)
	Expect(err).NotTo(HaveOccurred())
	Expect(toml.NewEncoder(configFile).Encode(&config)).To(Succeed())
	Expect(configFile.Close()).To(Succeed())

	cmd := exec.Command("containerd", "--config", configFile.Name())
	cmd.Env = append(os.Environ(), fmt.Sprintf("PATH=%s", fmt.Sprintf("%s:%s", os.Getenv("PATH"), "/usr/local/bin")))
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	return session
}

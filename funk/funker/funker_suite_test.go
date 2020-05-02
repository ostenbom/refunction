package funker_test

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFunker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Funker Suite")
}

func rcPath() string {
	return path.Join(os.Getenv("HOME"), ".funkrc")
}

var prevrc []byte

var _ = BeforeSuite(func() {
	funkrc, err := ioutil.ReadFile(rcPath())
	if err == nil {
		prevrc = funkrc
	} else {
		prevrc = []byte{}
	}

	os.RemoveAll(rcPath())
})

var _ = AfterSuite(func() {
	err := ioutil.WriteFile(rcPath(), prevrc, os.ModePerm)
	Expect(err).NotTo(HaveOccurred())
})

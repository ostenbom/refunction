package workerpool_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestWorkerpool(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Workerpool Suite")
}

package main_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestCri(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cri Suite")
}
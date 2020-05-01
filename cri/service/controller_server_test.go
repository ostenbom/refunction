package service_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/ostenbom/refunction/cri/service"
)

var _ = Describe("ControllerServer", func() {
	var c_server ControllerServer

	BeforeEach(func() {
		c_server = NewControllerServer()
		c_server.Start()
	})

	It("passes", func() {
		Expect(true).To(BeTrue())
	})
})

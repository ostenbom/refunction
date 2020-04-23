package controller_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/ostenbom/refunction/controller"
)

var _ = Describe("Controller", func() {
	var c Controller
	BeforeEach(func() {
		c = NewController()
	})

	It("returns an error if trying to attach without a pid", func() {
		Expect(c.Attach()).To(MatchError("controller has no pid"))
	})

	It("returns an error if trying to activate without streams", func() {
		c.SetPid(32769) // max_pid + 1
		Expect(c.Activate()).To(MatchError("controller has no in/out streams"))
	})

})

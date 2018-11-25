package worker_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/ostenbom/refunction/worker"
)

var _ = Describe("Worker", func() {

	It("has a started_at timestamp", func() {
		parent := NewWorker()

		Expect(parent.StartedAt).NotTo(BeNil())
	})

})

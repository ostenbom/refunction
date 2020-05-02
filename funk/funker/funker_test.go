package funker_test

import (
	"flag"
	"io/ioutil"
	"os"
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/urfave/cli/v2"

	refunction "github.com/ostenbom/refunction/cri/service/api/refunction/v1alpha"
	. "github.com/ostenbom/refunction/funk/funker"
	"github.com/ostenbom/refunction/funk/funker/funkerfakes"
)

var _ = Describe("Funker", func() {
	var ctx *cli.Context
	var funker *Funker
	var app *cli.App
	var fakeService *funkerfakes.FakeClient
	// var set *flag.FlagSet

	BeforeEach(func() {
		fakeService = new(funkerfakes.FakeClient)
		funker = NewFakeFunker(fakeService)
		app = funker.App()
	})

	Describe("SaveTarget", func() {
		It("saves the target url to .funkrc", func() {
			set := flag.NewFlagSet("unused", 0)
			set.String("url", "", "")
			Expect(set.Set("url", "localhost:7777")).To(Succeed())

			ctx = cli.NewContext(app, set, nil)
			err := funker.SaveTarget(ctx)
			Expect(err).NotTo(HaveOccurred())

			funkrc, err := ioutil.ReadFile(path.Join(os.Getenv("HOME"), ".funkrc"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(funkrc)).To(Equal("localhost:7777"))
		})
	})

	Context("when the target is saved", func() {
		BeforeEach(func() {
			set := flag.NewFlagSet("unused", 0)
			set.String("url", "", "")
			Expect(set.Set("url", "localhost:7777")).To(Succeed())

			ctx = cli.NewContext(app, set, nil)
			err := funker.SaveTarget(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("can send functions", func() {
			set := flag.NewFlagSet("unused", 0)
			set.String("container", "", "")
			Expect(set.Set("container", "potato")).To(Succeed())
			set.String("function", "", "")
			Expect(set.Set("function", "func: 1 + 1")).To(Succeed())

			fakeService.SendFunctionReturns(&refunction.FunctionResponse{}, nil)

			ctx = cli.NewContext(app, set, nil)
			err := funker.SendFunction(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeService.SendFunctionCallCount()).To(Equal(1))
			_, sendReq, _ := fakeService.SendFunctionArgsForCall(0)
			Expect(sendReq.Function).To(Equal("func: 1 + 1"))
			Expect(sendReq.ContainerId).To(Equal("potato"))
		})

		It("can send requests", func() {
			set := flag.NewFlagSet("unused", 0)
			set.String("container", "", "")
			Expect(set.Set("container", "potato")).To(Succeed())
			set.String("request", "", "")
			Expect(set.Set("request", "theargstring")).To(Succeed())

			fakeService.SendRequestReturns(&refunction.Response{Response: "imback"}, nil)

			ctx = cli.NewContext(app, set, nil)
			err := funker.SendRequest(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeService.SendRequestCallCount()).To(Equal(1))
			_, sendReq, _ := fakeService.SendRequestArgsForCall(0)
			Expect(sendReq.Request).To(Equal("theargstring"))
			Expect(sendReq.ContainerId).To(Equal("potato"))
		})

		It("can restore containers", func() {
			set := flag.NewFlagSet("unused", 0)
			set.String("container", "", "")
			Expect(set.Set("container", "potato")).To(Succeed())

			fakeService.RestoreReturns(&refunction.RestoreResponse{}, nil)

			ctx = cli.NewContext(app, set, nil)
			err := funker.Restore(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeService.RestoreCallCount()).To(Equal(1))
			_, restoreReq, _ := fakeService.RestoreArgsForCall(0)
			Expect(restoreReq.ContainerId).To(Equal("potato"))
		})
	})
})

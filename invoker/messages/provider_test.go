package messages_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/ostenbom/refunction/invoker/messages"
	"github.com/ostenbom/refunction/invoker/messages/messagesfakes"
)

var _ = Describe("Provider", func() {
	var (
		provider    MessageProvider
		producer    *messagesfakes.FakeProducer
		adminClient *messagesfakes.FakeAdminClient
		consumer    *messagesfakes.FakeConsumer
	)

	BeforeEach(func() {
		adminClient = new(messagesfakes.FakeAdminClient)
		producer = new(messagesfakes.FakeProducer)
		consumer = new(messagesfakes.FakeConsumer)

		provider = NewFakeProvider(adminClient, producer, consumer)
	})

	Context("when ensuring a topic exists", func() {
		It("calls CreateTopics with topic", func() {
			provider.EnsureTopic("anytopic")
			Expect(adminClient.CreateTopicsCallCount()).To(Equal(1))

			_, topicSpecs, _ := adminClient.CreateTopicsArgsForCall(0)
			Expect(len(topicSpecs)).To(Equal(1))
			Expect(topicSpecs[0].Topic).To(Equal("anytopic"))
		})

		It("calls CreateTopics with 1 partition and replication factor", func() {
			provider.EnsureTopic("anytopic")
			Expect(adminClient.CreateTopicsCallCount()).To(Equal(1))

			_, topicSpecs, _ := adminClient.CreateTopicsArgsForCall(0)
			Expect(len(topicSpecs)).To(Equal(1))
			Expect(topicSpecs[0].NumPartitions).To(Equal(1))
			Expect(topicSpecs[0].ReplicationFactor).To(Equal(1))
		})
	})
})

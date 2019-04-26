package messages_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/ostenbom/refunction/invoker/messages"
	"github.com/ostenbom/refunction/invoker/messages/messagesfakes"
)

var _ = Describe("Provider", func() {
	var (
		provider          MessageProvider
		connection        *messagesfakes.FakeKafkaConnection
		latestWriterHost  string
		latestWriterTopic string
		latestWriter      *messagesfakes.FakeWriter
	)

	BeforeEach(func() {
		connection = new(messagesfakes.FakeKafkaConnection)
		newWriterFunc := func(host string, topic string) Writer {
			latestWriterHost = host
			latestWriterTopic = topic
			latestWriter = new(messagesfakes.FakeWriter)
			return latestWriter
		}
		provider = NewFakeProvider(connection, newWriterFunc)
	})

	Describe("ensuring a topic exists", func() {
		It("calls CreateTopics with topic", func() {
			provider.EnsureTopic("anytopic")
			Expect(connection.CreateTopicsCallCount()).To(Equal(1))

			topicSpecs := connection.CreateTopicsArgsForCall(0)
			Expect(len(topicSpecs)).To(Equal(1))
			Expect(topicSpecs[0].Topic).To(Equal("anytopic"))
		})

		It("calls CreateTopics with 1 partition and replication factor", func() {
			provider.EnsureTopic("anytopic")
			Expect(connection.CreateTopicsCallCount()).To(Equal(1))

			topicSpecs := connection.CreateTopicsArgsForCall(0)
			Expect(len(topicSpecs)).To(Equal(1))
			Expect(topicSpecs[0].NumPartitions).To(Equal(1))
			Expect(topicSpecs[0].ReplicationFactor).To(Equal(1))
		})
	})

	Describe("writing messages", func() {
		It("creates a writer for the given topic", func() {
			Expect(provider.WriteMessage("pineapples", []byte{})).To(Succeed())
			Expect(latestWriterHost).To(Equal("anyhost"))
			Expect(latestWriterTopic).To(Equal("pineapples"))
			Expect(latestWriter).NotTo(BeNil())
		})

		It("writes the given value to the topic writer", func() {
			Expect(provider.WriteMessage("pineapples", []byte("pommegranite!"))).To(Succeed())
			Expect(latestWriter.WriteMessagesCallCount()).To(Equal(1))
			_, messages := latestWriter.WriteMessagesArgsForCall(0)
			Expect(len(messages)).NotTo(Equal(0))
			Expect(messages[0].Value).To(Equal([]byte("pommegranite!")))
		})
	})
})

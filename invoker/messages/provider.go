package messages

import (
	"fmt"

	"github.com/segmentio/kafka-go"
)

const defaultPartitions = 1
const defaultReplication = 1
const defaultNetwork = "tcp"

type MessageProvider interface {
	EnsureTopic(string) error
	Close() error
}

type messageProvider struct {
	kafkaConnection KafkaConnection
	writers         []Writer
	readers         []Reader
}

func NewMessageProvider(host string) (MessageProvider, error) {
	kafkaConnection, err := NewKafkaConnection(defaultNetwork, host)
	if err != nil {
		return nil, fmt.Errorf("could not start message provider: %s", err)
	}

	return messageProvider{
		kafkaConnection: kafkaConnection,
		writers:         []Writer{},
		readers:         []Reader{},
	}, nil
}

func NewFakeProvider(conn KafkaConnection) MessageProvider {
	return messageProvider{
		kafkaConnection: conn,
	}
}

func (p messageProvider) EnsureTopic(topic string) error {
	topicSpec := kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     defaultPartitions,
		ReplicationFactor: defaultReplication,
	}
	p.kafkaConnection.CreateTopics([]kafka.TopicConfig{topicSpec}...)
	return nil
}

func (p messageProvider) Close() error {
	return nil
}

package messages

import (
	"context"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

const defaultPartitions = 1
const defaultReplication = 1

type MessageProvider interface {
	EnsureTopic(string) error
	Close() error
}

type messageProvider struct {
	adminClient AdminClient
	producer    Producer
	consumer    Consumer
}

func NewMessageProvider(config *kafka.ConfigMap) (MessageProvider, error) {
	adminClient, err := NewAdminClient(config)
	if err != nil {
		return nil, fmt.Errorf("could not start message provider: %s", err)
	}

	producer, err := NewProducer(config)
	if err != nil {
		return nil, fmt.Errorf("could not start message provider: %s", err)
	}

	consumer, err := NewConsumer(config)
	if err != nil {
		return nil, fmt.Errorf("could not start message provider: %s", err)
	}

	return messageProvider{
		adminClient: adminClient,
		producer:    producer,
		consumer:    consumer,
	}, nil
}

func NewFakeProvider(admin AdminClient, producer Producer, consumer Consumer) MessageProvider {
	return messageProvider{
		adminClient: admin,
		producer:    producer,
		consumer:    consumer,
	}
}

func (p messageProvider) EnsureTopic(topic string) error {
	topicSpec := kafka.TopicSpecification{
		Topic:             topic,
		NumPartitions:     defaultPartitions,
		ReplicationFactor: defaultReplication,
	}
	p.adminClient.CreateTopics(context.Background(), []kafka.TopicSpecification{topicSpec})
	return nil
}

func (p messageProvider) Close() error {
	return nil
}

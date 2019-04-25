package messages

import (
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

//go:generate gobin -m -run github.com/maxbrunsfeld/counterfeiter/v6 . Consumer

type Consumer interface {
	Subscribe(string, kafka.RebalanceCb) error
	ReadMessage(time.Duration) (*kafka.Message, error)
}

type consumer struct {
	kafkaConsumer *kafka.Consumer
}

func NewConsumer(config *kafka.ConfigMap) (Consumer, error) {
	kafkaConsumer, err := kafka.NewConsumer(config)
	if err != nil {
		return nil, fmt.Errorf("could not start kafka admin client: %s", err)
	}
	return &consumer{
		kafkaConsumer: kafkaConsumer,
	}, nil
}

func (c consumer) Subscribe(topic string, rebalance kafka.RebalanceCb) error {
	return c.kafkaConsumer.Subscribe(topic, rebalance)
}

func (c consumer) ReadMessage(t time.Duration) (*kafka.Message, error) {
	return c.kafkaConsumer.ReadMessage(t)
}

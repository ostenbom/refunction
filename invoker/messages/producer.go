package messages

import (
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

//go:generate gobin -m -run github.com/maxbrunsfeld/counterfeiter/v6 . Producer

type Producer interface {
	Produce(*kafka.Message, chan kafka.Event) error
}

type producer struct {
	kafkaProducer *kafka.Producer
}

func NewProducer(config *kafka.ConfigMap) (Producer, error) {
	kafkaProducer, err := kafka.NewProducer(config)
	if err != nil {
		return nil, fmt.Errorf("could not start kafka admin client: %s", err)
	}
	return &producer{
		kafkaProducer: kafkaProducer,
	}, nil
}

func (p producer) Produce(m *kafka.Message, c chan kafka.Event) error {
	return p.kafkaProducer.Produce(m, c)
}

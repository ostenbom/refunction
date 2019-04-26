package messages

import (
	"fmt"

	"github.com/segmentio/kafka-go"
)

//go:generate gobin -m -run github.com/maxbrunsfeld/counterfeiter/v6 . KafkaConnection

type KafkaConnection interface {
	CreateTopics(...kafka.TopicConfig) error
	Close() error
}

type kafkaConnection struct {
	conn *kafka.Conn
}

func NewKafkaConnection(network string, host string) (KafkaConnection, error) {
	conn, err := kafka.Dial(network, host)
	if err != nil {
		return nil, fmt.Errorf("could not start kafka admin client: %s", err)
	}
	return &kafkaConnection{
		conn: conn,
	}, nil
}

func (c kafkaConnection) CreateTopics(topics ...kafka.TopicConfig) error {
	return c.conn.CreateTopics(topics...)
}

func (c kafkaConnection) Close() error {
	return c.conn.Close()
}

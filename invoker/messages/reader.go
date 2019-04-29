package messages

import (
	"context"

	"github.com/segmentio/kafka-go"
)

const defaultMinBytes = 10e3 // 10KB
const defaultMaxBytes = 10e6 // 10MB

//go:generate gobin -m -run github.com/maxbrunsfeld/counterfeiter/v6 . Reader

type Reader interface {
	ReadMessage(context.Context) (kafka.Message, error)
	Close() error
}

type reader struct {
	kafkaReader *kafka.Reader
}

type NewReaderFunc func(string, string) Reader

func NewReader(host string, topic string) Reader {
	kafkaReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{host},
		Topic:     topic,
		Partition: 0,
		MinBytes:  defaultMinBytes,
		MaxBytes:  defaultMaxBytes,
	})

	return &reader{
		kafkaReader: kafkaReader,
	}
}

func (c reader) ReadMessage(ctx context.Context) (kafka.Message, error) {
	return c.kafkaReader.ReadMessage(ctx)
}

func (c reader) Close() error {
	return c.Close()
}

package messages

import (
	"context"

	"github.com/segmentio/kafka-go"
)

//go:generate gobin -m -run github.com/maxbrunsfeld/counterfeiter/v6 . Writer

type Writer interface {
	WriteMessages(context.Context, ...kafka.Message) error
	Close() error
}

type writer struct {
	kafkaWriter *kafka.Writer
}

type NewWriterFunc func(string, string) Writer

func NewWriter(host string, topic string) Writer {
	kafkaWriter := kafka.NewWriter(kafka.WriterConfig{
		Brokers:   []string{host},
		Topic:     topic,
		BatchSize: 1,
	})
	return &writer{
		kafkaWriter: kafkaWriter,
	}
}

func (p writer) WriteMessages(ctx context.Context, messages ...kafka.Message) error {
	return p.kafkaWriter.WriteMessages(ctx, messages...)
}

func (p writer) Close() error {
	return p.kafkaWriter.Close()
}

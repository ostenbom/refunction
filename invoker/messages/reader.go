package messages

import (
	"context"
	"fmt"
	"time"

	"github.com/ostenbom/kafka-go"
	log "github.com/sirupsen/logrus"
)

const defaultMinBytes = 10
const defaultMaxBytes = 10e5 // 1MB

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
		Brokers:           []string{host},
		Topic:             topic,
		GroupID:           topic,
		Partition:         0,
		MinBytes:          defaultMinBytes,
		MaxBytes:          defaultMaxBytes,
		MaxWait:           time.Millisecond * 500,
		ReadLagInterval:   -1,
		HeartbeatInterval: time.Second * 10,
	})

	return &reader{
		kafkaReader: kafkaReader,
	}
}

func (c reader) ReadMessage(ctx context.Context) (kafka.Message, error) {
	return c.kafkaReader.ReadMessage(ctx)
}

func (c reader) Close() error {
	return c.kafkaReader.Close()
}

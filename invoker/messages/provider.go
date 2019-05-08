package messages

import (
	"context"
	"fmt"

	"github.com/ostenbom/kafka-go"
)

const defaultPartitions = 1
const defaultReplication = 1
const defaultNetwork = "tcp"

type MessageProvider interface {
	EnsureTopic(string) error
	WriteMessage(string, []byte) error
	ReadMessage(string) ([]byte, error)
	Close() error
}

type messageProvider struct {
	kafkaConnection KafkaConnection
	host            string
	writers         map[string]Writer
	readers         map[string]Reader
	newWriter       NewWriterFunc
	newReader       NewReaderFunc
}

func NewMessageProvider(host string) (MessageProvider, error) {
	kafkaConnection, err := NewKafkaConnection(defaultNetwork, host)
	if err != nil {
		return nil, fmt.Errorf("could not start message provider: %s", err)
	}

	return messageProvider{
		kafkaConnection: kafkaConnection,
		host:            host,
		writers:         make(map[string]Writer),
		readers:         make(map[string]Reader),
		newWriter:       NewWriter,
		newReader:       NewReader,
	}, nil
}

func NewFakeProvider(conn KafkaConnection, writerFunc NewWriterFunc, readerFunc NewReaderFunc) MessageProvider {
	return messageProvider{
		kafkaConnection: conn,
		host:            "anyhost",
		writers:         make(map[string]Writer),
		readers:         make(map[string]Reader),
		newWriter:       writerFunc,
		newReader:       readerFunc,
	}
}

func (p messageProvider) EnsureTopic(topic string) error {
	topicSpec := kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     defaultPartitions,
		ReplicationFactor: defaultReplication,
	}
	return p.kafkaConnection.CreateTopics([]kafka.TopicConfig{topicSpec}...)
}

func (p messageProvider) WriteMessage(topic string, value []byte) error {
	writer, exists := p.writers[topic]
	if !exists {
		writer = p.newWriter(p.host, topic)
		p.writers[topic] = writer
	}
	msg := kafka.Message{
		Value: value,
	}

	return writer.WriteMessages(context.Background(), []kafka.Message{msg}...)
}

func (p messageProvider) ReadMessage(topic string) ([]byte, error) {
	reader, exists := p.readers[topic]
	if !exists {
		reader = p.newReader(p.host, topic)
		p.readers[topic] = reader
	}
	msg, err := reader.ReadMessage(context.Background())
	if err != nil {
		return nil, fmt.Errorf("could not read message from topic %s: %s", topic, err)
	}
	return msg.Value, nil
}

func (p messageProvider) Close() error {
	for k := range p.writers {
		err := p.writers[k].Close()
		if err != nil {
			return err
		}
	}

	for k := range p.readers {
		err := p.readers[k].Close()
		if err != nil {
			return err
		}
	}

	return p.kafkaConnection.Close()
}

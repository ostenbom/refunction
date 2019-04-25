package messages

import (
	"context"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

//go:generate gobin -m -run github.com/maxbrunsfeld/counterfeiter/v6 . AdminClient

type AdminClient interface {
	CreateTopics(context.Context, []kafka.TopicSpecification, ...kafka.CreateTopicsAdminOption) ([]kafka.TopicResult, error)
}

type adminClient struct {
	kafkaAdmin *kafka.AdminClient
}

func NewAdminClient(config *kafka.ConfigMap) (AdminClient, error) {
	kafkaAdmin, err := kafka.NewAdminClient(config)
	if err != nil {
		return nil, fmt.Errorf("could not start kafka admin client: %s", err)
	}
	return &adminClient{
		kafkaAdmin: kafkaAdmin,
	}, nil
}

func (a adminClient) CreateTopics(ctx context.Context, topicSpecs []kafka.TopicSpecification, adminOpts ...kafka.CreateTopicsAdminOption) ([]kafka.TopicResult, error) {
	return a.kafkaAdmin.CreateTopics(ctx, topicSpecs, adminOpts...)
}

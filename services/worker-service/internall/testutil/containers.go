package testutil

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/kafka"
	"github.com/testcontainers/testcontainers-go/modules/minio"
)

type TestEnv struct {
	Kafka *kafka.KafkaContainer
	Minio *minio.MinioContainer
}

func SetupIntegrationEnv(ctx context.Context) (*TestEnv, error) {
	// 1. Kafka
	kafkaContainer, err := kafka.Run(ctx,
		"confluentinc/confluent-local:7.5.0",
		kafka.WithClusterID("test-cluster"),
		testcontainers.WithEnv(map[string]string{
			"KAFKA_AUTO_CREATE_TOPICS_ENABLE": "true",
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start kafka: %w", err)
	}

	// 2. Minio
	minioContainer, err := minio.Run(ctx, "minio/minio:latest")
	if err != nil {
		return nil, fmt.Errorf("failed to start minio: %w", err)
	}

	return &TestEnv{
		Kafka: kafkaContainer,
		Minio: minioContainer,
	}, nil
}

func (e *TestEnv) Terminate(ctx context.Context) {
	if e.Kafka != nil {
		e.Kafka.Terminate(ctx)
	}
	if e.Minio != nil {
		e.Minio.Terminate(ctx)
	}
}

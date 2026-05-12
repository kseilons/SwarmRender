package testutil

import (
	"context"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/kafka"
	"github.com/testcontainers/testcontainers-go/modules/minio"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type IntegrationEnv struct {
	Postgres *postgres.PostgresContainer
	Kafka    *kafka.KafkaContainer
	Minio    *minio.MinioContainer
}

func SetupIntegrationEnv(ctx context.Context) (*IntegrationEnv, error) {
	// 1. Postgres
	pgContainer, err := postgres.Run(ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("jobdb"),
		postgres.WithUsername("user"),
		postgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(10*time.Second)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres: %w", err)
	}

	// 2. Kafka
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

	// 3. Minio
	minioContainer, err := minio.Run(ctx, "minio/minio:latest")
	if err != nil {
		return nil, fmt.Errorf("failed to start minio: %w", err)
	}

	return &IntegrationEnv{
		Postgres: pgContainer,
		Kafka:    kafkaContainer,
		Minio:    minioContainer,
	}, nil
}

func (e *IntegrationEnv) Terminate(ctx context.Context) {
	if e.Postgres != nil {
		e.Postgres.Terminate(ctx)
	}
	if e.Kafka != nil {
		e.Kafka.Terminate(ctx)
	}
	if e.Minio != nil {
		e.Minio.Terminate(ctx)
	}
}

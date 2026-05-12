//go:build integration

package integration

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"testing"
	"time"

	"kseilons/job-service/internall/app"
	"kseilons/job-service/internall/config"
	"kseilons/job-service/internall/testutil"

	"github.com/golang-jwt/jwt/v5"
	"github.com/segmentio/kafka-go"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type JobIntegrationTestSuite struct {
	suite.Suite
	env         *testutil.IntegrationEnv
	app         *app.App
	cancel      context.CancelFunc
	baseURL     string
	kafkaWriter *kafka.Writer
}

func (s *JobIntegrationTestSuite) SetupSuite() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	s.cancel = cancel

	// 1. Setup Containers
	env, err := testutil.SetupIntegrationEnv(ctx)
	require.NoError(s.T(), err)
	s.env = env

	// 2. Prepare Config
	kafkaBrokers, err := s.env.Kafka.Brokers(ctx)
	require.NoError(s.T(), err)

	minioConn, err := s.env.Minio.ConnectionString(ctx)
	require.NoError(s.T(), err)
	// MinIO connection string usually contains http://, let's strip it for config if needed
	minioHost := minioConn
	if idx := 7; len(minioHost) > idx && minioHost[:idx] == "http://" {
		minioHost = minioHost[idx:]
	}

	pgHost, _ := s.env.Postgres.Host(ctx)
	pgPort, _ := s.env.Postgres.MappedPort(ctx, "5432")
	pgPortInt, _ := strconv.Atoi(pgPort.Port())

	cfg := &config.Config{
		Env: "local",
		Http: config.HttpServerConfig{
			Address: ":8081",
		},
		Database: config.DatabaseConfig{
			Host:     pgHost,
			Port:     pgPortInt,
			User:     "user",
			Password: "password",
			Name:     "jobdb",
			SSLMode:  "disable",
		},
		Kafka: config.KafkaConfig{
			Brokers:       kafkaBrokers,
			RenderTopic:    "render-tasks",
			MetadataTopic:  "metadata-tasks",
			StitchingTopic: "stitching-tasks",
			ResultTopic:    "render-results",
			ResultGroupID:  "test-group",
		},
		Minio: config.MinioConfig{
			Endpoint:        minioHost,
			AccessKey:       "minioadmin",
			SecretKey:       "minioadmin",
			Bucket:          "render-bucket",
			UseSSL:          false,
			PresignedExpiry: 3600,
		},
	}
	// Run migrations
	err = testutil.RunMigrations(ctx, cfg.Database.URL(), "../../migrations")
	require.NoError(s.T(), err)

	// 3. Setup Kafka Topics for Testing
	client := &kafka.Client{
		Addr: kafka.TCP(kafkaBrokers...),
	}

	topicConfigs := []kafka.TopicConfig{
		{Topic: cfg.Kafka.RenderTopic, NumPartitions: 1, ReplicationFactor: 1},
		{Topic: cfg.Kafka.MetadataTopic, NumPartitions: 1, ReplicationFactor: 1},
		{Topic: cfg.Kafka.StitchingTopic, NumPartitions: 1, ReplicationFactor: 1},
		{Topic: cfg.Kafka.ResultTopic, NumPartitions: 1, ReplicationFactor: 1},
		{Topic: "render-tasks-cpu", NumPartitions: 1, ReplicationFactor: 1},
	}
	
	_, err = client.CreateTopics(ctx, &kafka.CreateTopicsRequest{
		Topics: topicConfigs,
	})
	require.NoError(s.T(), err)
	
	// Give Kafka some time to propagate metadata
	time.Sleep(5 * time.Second)

	// 4. Setup MinIO Bucket for Testing
	minioClient, err := minio.New(cfg.Minio.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.Minio.AccessKey, cfg.Minio.SecretKey, ""),
		Secure: cfg.Minio.UseSSL,
	})
	require.NoError(s.T(), err)
	
	err = minioClient.MakeBucket(ctx, cfg.Minio.Bucket, minio.MakeBucketOptions{})
	if err != nil {
		// Ignore if bucket already exists
		exists, _ := minioClient.BucketExists(ctx, cfg.Minio.Bucket)
		if !exists {
			require.NoError(s.T(), err)
		}
	}

	// 5. Setup Kafka Writer for Testing
	s.kafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP(kafkaBrokers...),
		Topic:    cfg.Kafka.ResultTopic,
		Balancer: &kafka.LeastBytes{},
	}

	// 4. Start App
	s.app = app.New(cfg, slog.Default())
	err = s.app.Init(ctx)
	require.NoError(s.T(), err)

	go func() {
		if err := s.app.Run(ctx); err != nil {
			fmt.Printf("App run error: %v\n", err)
		}
	}()

	time.Sleep(2 * time.Second)
	s.baseURL = "http://localhost:8081"
}

func (s *JobIntegrationTestSuite) TearDownSuite() {
	ctx := context.Background()
	if s.cancel != nil {
		s.cancel()
	}
	if s.kafkaWriter != nil {
		s.kafkaWriter.Close()
	}
	if s.env != nil {
		s.env.Terminate(ctx)
	}
}

func (s *JobIntegrationTestSuite) createToken(userID string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   userID,
		"email": userID + "@example.com",
		"realm_access": map[string]interface{}{
			"roles": []string{"user"},
		},
	})
	tokenString, _ := token.SignedString([]byte("secret"))
	return tokenString
}

func stringPtr(s string) *string { return &s }

func TestJobIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(JobIntegrationTestSuite))
}

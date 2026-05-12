//go:build integration

package integration

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"kseilons/worker-service/internall/app"
	"kseilons/worker-service/internall/config"
	"kseilons/worker-service/internall/testutil"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type WorkerIntegrationTestSuite struct {
	suite.Suite
	env         *testutil.TestEnv
	app         *app.App
	blenderPath string
}

func (s *WorkerIntegrationTestSuite) SetupSuite() {
	ctx := context.Background()

	// 1. Setup Infrastructure
	env, err := testutil.SetupIntegrationEnv(ctx)
	require.NoError(s.T(), err)
	s.env = env

	kafkaBrokers, err := env.Kafka.Brokers(ctx)
	require.NoError(s.T(), err)

	minioHost, err := env.Minio.Host(ctx)
	require.NoError(s.T(), err)
	minioPort, err := env.Minio.MappedPort(ctx, "9000")
	require.NoError(s.T(), err)
	minioEndpoint := fmt.Sprintf("%s:%s", minioHost, minioPort.Port())

	// 2. Compile Fake Blender
	s.blenderPath = filepath.Join(os.TempDir(), "fake_blender")
	if os.PathSeparator == '\\' {
		s.blenderPath += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", s.blenderPath, "./fake_blender/main.go")
	err = cmd.Run()
	require.NoError(s.T(), err)

	// 3. Setup MinIO Bucket
	mClient, err := minio.New(minioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4("minioadmin", "minioadmin", ""),
		Secure: false,
	})
	require.NoError(s.T(), err)
	err = mClient.MakeBucket(ctx, "render-bucket", minio.MakeBucketOptions{})
	require.NoError(s.T(), err)

	// Upload dummy scene.blend
	dummyContent := "dummy blender file content"
	_, err = mClient.PutObject(ctx, "render-bucket", "scene.blend", bytes.NewReader([]byte(dummyContent)), int64(len(dummyContent)), minio.PutObjectOptions{})
	require.NoError(s.T(), err)

	// 3.5 Create Kafka Topics
	kafkaClient := &kafka.Client{
		Addr: kafka.TCP(kafkaBrokers...),
	}
	_, err = kafkaClient.CreateTopics(ctx, &kafka.CreateTopicsRequest{
		Topics: []kafka.TopicConfig{
			{Topic: "metadata-tasks", NumPartitions: 1, ReplicationFactor: 1},
			{Topic: "render-tasks-cpu", NumPartitions: 1, ReplicationFactor: 1},
			{Topic: "render-results", NumPartitions: 1, ReplicationFactor: 1},
		},
	})
	require.NoError(s.T(), err)
	time.Sleep(2 * time.Second) // Wait for metadata propagation

	// 4. Setup App Config
	cfg := &config.Config{
		Env: "test",
		Kafka: config.KafkaConfig{
			Brokers:       kafkaBrokers,
			RenderTopic:   "render-tasks",
			MetadataTopic: "metadata-tasks",
			ResultTopic:   "render-results",
			GroupID:       "worker-test-group",
		},
		Minio: config.MinioConfig{
			Endpoint:  minioEndpoint,
			AccessKey: "minioadmin",
			SecretKey: "minioadmin",
			Bucket:    "render-bucket",
			UseSSL:    false,
		},
		Blender: config.BlenderConfig{
			Path:           s.blenderPath,
			AllowedDevices: []string{"CPU"},
			Threads:        1,
		},
	}

	// 5. Start App
	s.app = app.New(cfg, slog.Default())
	appCtx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := s.app.Run(appCtx); err != nil {
			s.T().Logf("app run error: %v", err)
		}
	}()
	s.T().Cleanup(cancel)

	// Wait for health check
	require.Eventually(s.T(), func() bool {
		resp, err := http.Get("http://localhost:8081/health")
		return err == nil && resp.StatusCode == http.StatusOK
	}, 15*time.Second, 500*time.Millisecond)
}

func (s *WorkerIntegrationTestSuite) TearDownSuite() {
	ctx := context.Background()
	if s.app != nil {
		s.app.Shutdown()
	}
	if s.env != nil {
		s.env.Terminate(ctx)
	}
	os.Remove(s.blenderPath)
}

func TestWorkerIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(WorkerIntegrationTestSuite))
}

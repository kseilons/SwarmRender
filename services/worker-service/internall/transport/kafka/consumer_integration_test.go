//go:build integration

package kafka_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/mock"
	"kseilons/worker-service/internall/config"
	"kseilons/worker-service/internall/domain"
	transport "kseilons/worker-service/internall/transport/kafka"
	kafkamocks "kseilons/worker-service/internall/transport/kafka/mocks"
)

func TestConsumer_Integration(t *testing.T) {
	// 1. Setup config from file
	allCfg, err := config.Load("../../../config/config-test.yaml")
	if err != nil {
		t.Fatalf("failed to load test config: %v", err)
	}
	cfg := allCfg.Kafka

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	mockUC := kafkamocks.NewRenderUseCase(t)

	// 2. Prepare message to send
	task := domain.RenderTask{
		ProjectID: "kafka-proj-123",
		BlendFile: "test.blend",
		CommonTask: domain.CommonTask{
			StartFrame: 1,
			EndFrame:   1,
		},
	}
	payload, _ := json.Marshal(task)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 3. Create topic if not exists
	conn, _ := kafka.Dial("tcp", cfg.Brokers[0])
	if conn != nil {
		controller, _ := conn.Controller()
		var controllerConn *kafka.Conn
		controllerConn, _ = kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
		if controllerConn != nil {
			_ = controllerConn.CreateTopics(kafka.TopicConfig{
				Topic:             cfg.Topic,
				NumPartitions:     1,
				ReplicationFactor: 1,
			})
			_ = controllerConn.Close()
		}
		_ = conn.Close()
	}

	// 4. Send message to Kafka
	writer := &kafka.Writer{
		Addr:     kafka.TCP(cfg.Brokers...),
		Topic:    cfg.Topic,
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	err = writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte("test-key"),
		Value: payload,
	})
	if err != nil {
		t.Fatalf("failed to write message to kafka: %v", err)
	}

	// 4. Set expectations
	processed := make(chan bool, 1)
	mockUC.On("Execute", mock.Anything, mock.Anything, mock.MatchedBy(func(t domain.RenderTask) bool {
		return t.ProjectID == "kafka-proj-123"
	})).Return([]byte("ok"), nil).Run(func(args mock.Arguments) {
		processed <- true
	})

	// 5. Run Consumer
	consumer := transport.NewConsumer(cfg, mockUC, logger)
	defer consumer.Close()

	// Run in a goroutine
	go func() {
		_ = consumer.Run(ctx)
	}()

	// 6. Wait for success or timeout
	select {
	case <-ctx.Done():
		t.Errorf("test timed out waiting for consumer to process message")
	case <-processed:
		// Success!
		mockUC.AssertExpectations(t)
	}

	cancel() // Stop consumer
}

package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"kseilons/worker-service/internall/domain"

	"github.com/segmentio/kafka-go"
)

type ResultProducer struct {
	writer *kafka.Writer
	logger *slog.Logger
}

func NewResultProducer(brokers []string, topic string, logger *slog.Logger) *ResultProducer {
	return &ResultProducer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
		},
		logger: logger,
	}
}

func (p *ResultProducer) ReportResult(ctx context.Context, result domain.RenderResult) error {
	payload, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal render result: %w", err)
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(result.TaskID),
		Value: payload,
	})
	if err != nil {
		return fmt.Errorf("failed to write result to kafka: %w", err)
	}

	p.logger.Info("reported task result to kafka",
		slog.String("task_id", result.TaskID),
		slog.String("status", string(result.Status)),
	)

	return nil
}

func (p *ResultProducer) Close() error {
	return p.writer.Close()
}

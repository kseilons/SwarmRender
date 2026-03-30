package kafka

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/segmentio/kafka-go"
	"kseilons/job-service/internall/domain"
	"kseilons/job-service/pkg/logger"
)

type JobUseCase interface {
	UpdateJobStatus(ctx context.Context, id string, status domain.Status, logger *slog.Logger) error
}

type Consumer struct {
	reader  *kafka.Reader
	useCase JobUseCase
	logger  *slog.Logger
}

func NewConsumer(brokers []string, topic, groupID string, useCase JobUseCase, logger *slog.Logger) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: groupID,
		}),
		useCase: useCase,
		logger:  logger,
	}
}

func (c *Consumer) Run(ctx context.Context) error {
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			c.logger.Error("failed to read message", logger.Err(err))
			continue
		}

		c.logger.Info("received result message",
			slog.String("key", string(msg.Key)),
			slog.Int("partition", msg.Partition),
			slog.Int64("offset", msg.Offset),
		)

		var result domain.RenderResult
		if err := json.Unmarshal(msg.Value, &result); err != nil {
			c.logger.Error("failed to unmarshal result",
				slog.String("key", string(msg.Key)),
				logger.Err(err),
			)
			continue
		}

		c.logger.Info("processing result",
			slog.String("task_id", result.TaskID),
			slog.String("status", string(result.Status)),
			slog.String("worker_id", result.WorkerID),
		)

		// Example: if status is FAILED, maybe update job status
		// Here we need logic to find JobID from TaskID or ProjectID
		// For now, let's assume TaskID has prefix or we have a map
		// But let's just log and update if it's a metadata task result
		if result.Metadata != nil {
			c.logger.Info("received metadata",
				slog.Int("frame_start", result.Metadata.FrameStart),
				slog.Int("frame_end", result.Metadata.FrameEnd),
			)
		}

		if result.Error != "" {
			c.logger.Warn("result contains error",
				slog.String("task_id", result.TaskID),
				slog.String("error", result.Error),
			)
		}

		// (Actual logic for updating the DB would go here)
		// For now, just logging the intention
		c.logger.Info("successfully processed result", slog.String("task_id", result.TaskID))
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

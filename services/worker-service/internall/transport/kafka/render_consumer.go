package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"time"

	"kseilons/worker-service/internall/config"
	"kseilons/worker-service/internall/domain"

	"github.com/segmentio/kafka-go"
)

type RenderUseCase interface {
	Execute(ctx context.Context, logger *slog.Logger, task domain.RenderTask) ([]byte, error)
}

type MetadataUseCase interface {
	Execute(ctx context.Context, logger *slog.Logger, task domain.MetadataTask) error
}

type Reader interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

type Consumer struct {
	reader     Reader
	renderUC   RenderUseCase
	metadataUC MetadataUseCase
	logger     *slog.Logger
}

func NewConsumer(cfg config.KafkaConfig, renderUC RenderUseCase, metadataUC MetadataUseCase, logger *slog.Logger, allowedDevices []string) *Consumer {
	topics := []string{cfg.MetadataTopic}
	if cfg.MetadataTopic == "" {
		topics = []string{"metadata-tasks"}
	}

	for _, device := range allowedDevices {
		topic := fmt.Sprintf("%s-%s", cfg.RenderTopic, strings.ToLower(device))
		topics = append(topics, topic)
	}

	logger.Info("subscribing to topics", slog.Any("topics", topics))

	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:     cfg.Brokers,
			GroupTopics: topics,
			GroupID:     cfg.GroupID,
			MinBytes:    1,
			MaxBytes:    10e6, // 10MB
			MaxWait:     1 * time.Second,
			StartOffset: kafka.FirstOffset,
		}),
		renderUC:   renderUC,
		metadataUC: metadataUC,
		logger:     logger,
	}
}

func (c *Consumer) Run(ctx context.Context) error {
	c.logger.Info("starting kafka consumer")

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			c.logger.Error("failed to fetch message from kafka", slog.Any("error", err))
			continue
		}

		c.logger.Info("received message from kafka",
			slog.String("topic", msg.Topic),
			slog.String("key", string(msg.Key)),
			slog.Int("partition", msg.Partition),
			slog.Int64("offset", msg.Offset),
		)

		var taskErr error
		switch {
		case msg.Topic == "" || msg.Topic == "metadata-tasks": // Handle both for safety
			var task domain.MetadataTask
			if err := json.Unmarshal(msg.Value, &task); err != nil {
				c.logger.Error("failed to unmarshal metadata task", slog.Any("error", err))
				break
			}
			taskLog := c.logger.With(task.LogAttrs()...)
			taskLog.Info("processing metadata task")
			taskErr = c.metadataUC.Execute(ctx, taskLog, task)
			if taskErr != nil {
				taskLog.Error("failed to execute metadata task", slog.Any("error", taskErr))
			} else {
				taskLog.Info("metadata task completed successfully")
			}

		case strings.HasPrefix(msg.Topic, "render-tasks"):
			var task domain.RenderTask
			if err := json.Unmarshal(msg.Value, &task); err != nil {
				c.logger.Error("failed to unmarshal render task", slog.Any("error", err))
				break
			}
			taskLog := c.logger.With(task.LogAttrs()...)
			taskLog.Info("processing render task", slog.String("topic", msg.Topic))
			_, taskErr = c.renderUC.Execute(ctx, taskLog, task)
			if taskErr != nil {
				taskLog.Error("failed to execute render task", slog.Any("error", taskErr))
			} else {
				taskLog.Info("render task completed successfully")
			}

		default:
			c.logger.Error("received message from unknown topic", slog.String("topic", msg.Topic))
		}

		if commitErr := c.reader.CommitMessages(ctx, msg); commitErr != nil {
			c.logger.Error("failed to commit message after processing", slog.Any("error", commitErr))
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

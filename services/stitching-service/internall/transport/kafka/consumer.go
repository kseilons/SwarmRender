package kafka

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/segmentio/kafka-go"
	"kseilons/stitching-service/internall/domain"
)

type StitchingUseCase interface {
	Execute(ctx context.Context, logger *slog.Logger, task domain.StitchingTask) error
}

type Consumer struct {
	reader  *kafka.Reader
	useCase StitchingUseCase
	logger  *slog.Logger
}

func NewConsumer(brokers []string, topic, groupID string, useCase StitchingUseCase, logger *slog.Logger) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})

	return &Consumer{
		reader:  reader,
		useCase: useCase,
		logger:  logger,
	}
}

func (c *Consumer) Run(ctx context.Context) error {
	c.logger.Info("stitching consumer started", slog.String("topic", c.reader.Config().Topic))

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			c.logger.Error("failed to fetch message", slog.Any("error", err))
			continue
		}

		var task domain.StitchingTask
		if err := json.Unmarshal(msg.Value, &task); err != nil {
			c.logger.Error("failed to unmarshal task", slog.Any("error", err))
			c.reader.CommitMessages(ctx, msg)
			continue
		}

		c.logger.Info("processing stitching task", slog.Any("task", task.LogAttrs()))

		if err := c.useCase.Execute(ctx, c.logger, task); err != nil {
			c.logger.Error("failed to execute stitching task", slog.Any("error", err))
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.logger.Error("failed to commit message", slog.Any("error", err))
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

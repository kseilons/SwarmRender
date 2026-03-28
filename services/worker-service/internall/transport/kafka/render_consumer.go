package kafka

import (
	"context"
	"encoding/json"
	"log/slog"

	"kseilons/worker-service/internall/config"
	"kseilons/worker-service/internall/domain"

	"github.com/segmentio/kafka-go"
)

type RenderUseCase interface {
	Execute(ctx context.Context, logger *slog.Logger, task domain.RenderTask) ([]byte, error)
}

type Reader interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

type Consumer struct {
	reader  Reader
	useCase RenderUseCase
	logger  *slog.Logger
}

func NewConsumer(cfg config.KafkaConfig, useCase RenderUseCase, logger *slog.Logger) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  cfg.Brokers,
			Topic:    cfg.Topic,
			GroupID:  cfg.GroupID,
			MinBytes: 10e3, // 10KB
			MaxBytes: 10e6, // 10MB
		}),
		useCase: useCase,
		logger:  logger,
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
			slog.String("key", string(msg.Key)),
			slog.Int("partition", msg.Partition),
			slog.Int64("offset", msg.Offset),
		)

		var task domain.RenderTask
		if err := json.Unmarshal(msg.Value, &task); err != nil {
			c.logger.Error("failed to unmarshal render task",
				slog.Any("error", err),
				slog.String("payload", string(msg.Value)),
			)
			if commitErr := c.reader.CommitMessages(ctx, msg); commitErr != nil {
				c.logger.Error("failed to commit message after unmarshal error", slog.Any("error", commitErr))
			}
			continue
		}

		// Use LogAttrs for contextual logging
		taskLog := c.logger.With(task.LogAttrs()...)
		taskLog.Info("processing task from kafka")

		_, err = c.useCase.Execute(ctx, taskLog, task)
		if err != nil {
			taskLog.Error("failed to execute render task", slog.Any("error", err))
			// Here we could handle retries or dead letters
		} else {
			taskLog.Info("task from kafka completed successfully")
		}

		if commitErr := c.reader.CommitMessages(ctx, msg); commitErr != nil {
			taskLog.Error("failed to commit message after processing", slog.Any("error", commitErr))
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

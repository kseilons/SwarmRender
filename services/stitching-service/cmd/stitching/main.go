package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"kseilons/stitching-service/internall/config"
	"kseilons/stitching-service/internall/repository"
	"kseilons/stitching-service/internall/transport/kafka"
	"kseilons/stitching-service/internall/usecase"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.MustLoad()

	// Initialize Repositories
	minioRepo, err := repository.NewMinioRepository(cfg.Minio)
	if err != nil {
		logger.Error("failed to initialize minio repository", slog.Any("error", err))
		os.Exit(1)
	}

	// Initialize Producers
	producer := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.ResultTopic)
	defer producer.Close()

	// Initialize Stitcher
	stitcher := usecase.NewFFmpegStitcher()

	// Initialize UseCases
	stitchingUC := usecase.NewStitchingUseCase(minioRepo, producer, stitcher)

	// Initialize Consumers
	consumer := kafka.NewConsumer(
		cfg.Kafka.Brokers,
		cfg.Kafka.InputTopic,
		cfg.Kafka.ConsumerGroup,
		stitchingUC,
		logger,
	)
	defer consumer.Close()

	// Handle Graceful Shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Info("received signal, shutting down", slog.String("signal", sig.String()))
		cancel()
	}()

	// Start Consumer
	if err := consumer.Run(ctx); err != nil {
		logger.Error("consumer error", slog.Any("error", err))
	}
}

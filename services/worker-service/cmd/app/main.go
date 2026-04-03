package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"kseilons/worker-service/internall/config"
	"kseilons/worker-service/internall/render"
	"kseilons/worker-service/internall/repository"
	"kseilons/worker-service/internall/transport/kafka"
	"kseilons/worker-service/internall/usecase"
	"kseilons/worker-service/pkg/logger"
)

func main() {
	cfg := config.MustLoad()

	log := logger.SetupLogger(cfg.Env)
	log.Info("starting application", "env", cfg.Env)

	// Context for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	engine := render.NewEngine(cfg.Blender.Path)

	minioRepo, err := repository.NewMinioRepository(cfg.Minio)
	if err != nil {
		log.Error("failed to initialize minio repository", logger.Err(err))
		os.Exit(1)
	}

	resultProducer := kafka.NewResultProducer(cfg.Kafka.Brokers, cfg.Kafka.ResultTopic, log)
	defer func() {
		if err := resultProducer.Close(); err != nil {
			log.Error("failed to close kafka result producer", logger.Err(err))
		}
	}()

	renderUseCase := usecase.NewRenderUseCase(engine, minioRepo, resultProducer, cfg.Blender.AllowedDevices, cfg.Blender.Threads)
	metadataUseCase := usecase.NewMetadataUseCase(engine, minioRepo, resultProducer)

	// Initialize Transport (Kafka Consumer)
	consumer := kafka.NewConsumer(cfg.Kafka, renderUseCase, metadataUseCase, log, cfg.Blender.AllowedDevices)
	defer func() {
		if err := consumer.Close(); err != nil {
			log.Error("failed to close kafka consumer", logger.Err(err))
		}
	}()

	// Start consumer in a goroutine
	go func() {
		if err := consumer.Run(ctx); err != nil {
			log.Error("kafka consumer failed", logger.Err(err))
			os.Exit(1)
		}
	}()

	log.Info("worker service is running and waiting for tasks from kafka...")

	// Wait for termination signal
	<-ctx.Done()
	log.Info("shutting down application gracefully")

	fmt.Println("application stopped")
}

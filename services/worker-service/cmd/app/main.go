package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"kseilons/worker-service/internall/config"
	"kseilons/worker-service/internall/render"
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

	renderUseCase := usecase.NewRenderUseCase(engine, cfg.Blender.AllowedDevices)

	// Initialize Transport (Kafka Consumer)
	consumer := kafka.NewConsumer(cfg.Kafka, renderUseCase, log)
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

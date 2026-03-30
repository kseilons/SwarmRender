package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kseilons/job-service/internall/config"
	"kseilons/job-service/internall/producer"
	"kseilons/job-service/internall/repository/postgres"
	transport_http "kseilons/job-service/internall/transport/http"
	transport_kafka "kseilons/job-service/internall/transport/kafka"
	"kseilons/job-service/internall/usecase"
	"kseilons/job-service/pkg/logger"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Config & Context
	cfg := config.MustLoad()

	// Context for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Logger
	log := logger.SetupLogger(cfg.Env)
	slog.SetDefault(log)

	log.Info("starting job-service", slog.String("env", cfg.Env))

	// Database
	pool, err := pgxpool.New(ctx, cfg.Database.URL())
	if err != nil {
		log.Error("failed to connect to postgres", logger.Err(err))
		os.Exit(1)
	}
	defer pool.Close()

	// Layers
	jobRepo := postgres.NewJobRepository(pool)
	taskRepo := postgres.NewTaskRepository(pool)
	prod := producer.NewKafkaProducer(cfg.Kafka.Brokers)
	defer prod.Close()

	hub := usecase.NewHub()
	go hub.Run()

	jobUseCase := usecase.NewJobUseCase(jobRepo, taskRepo, prod, cfg.Kafka, hub)

	// Result Consumer
	consumer := transport_kafka.NewConsumer(
		cfg.Kafka.Brokers,
		cfg.Kafka.ResultTopic,
		cfg.Kafka.ResultGroupID,
		jobUseCase,
		log,
	)

	// Start consumer in background
	go func() {
		if err := consumer.Run(ctx); err != nil {
			log.Error("result consumer error", logger.Err(err))
		}
	}()

	// HTTP Server settings
	gin.SetMode(gin.ReleaseMode)
	if cfg.Env == "local" {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.Default()
	router.Use(cors.Default()) // Enable CORS
	handler := transport_http.NewHandler(jobUseCase, hub)
	handler.Register(router)

	srv := &http.Server{
		Addr:    cfg.Http.Address,
		Handler: router,
	}

	go func() {
		log.Info("http server starting", slog.String("address", cfg.Http.Address))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server failed", logger.Err(err))
		}
	}()

	<-ctx.Done()

	log.Info("shutting down job-service gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("forced server shutdown", logger.Err(err))
	}

	log.Info("job-service exited")
}

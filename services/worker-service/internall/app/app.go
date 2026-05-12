package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"kseilons/worker-service/internall/config"
	"kseilons/worker-service/internall/render"
	"kseilons/worker-service/internall/repository"
	"kseilons/worker-service/internall/transport/kafka"
	"kseilons/worker-service/internall/usecase"
	"github.com/gin-gonic/gin"
	prometheus "github.com/zsais/go-gin-prometheus"
)

type App struct {
	cfg      *config.Config
	log      *slog.Logger
	consumer *kafka.Consumer
	httpSrv  *http.Server
}

func New(cfg *config.Config, log *slog.Logger) *App {
	return &App{
		cfg: cfg,
		log: log,
	}
}

func (a *App) Run(ctx context.Context) error {
	engine := render.NewEngine(a.cfg.Blender.Path)

	minioRepo, err := repository.NewMinioRepository(a.cfg.Minio)
	if err != nil {
		return fmt.Errorf("failed to initialize minio repository: %w", err)
	}

	resultProducer := kafka.NewResultProducer(a.cfg.Kafka.Brokers, a.cfg.Kafka.ResultTopic, a.log)
	
	renderUseCase := usecase.NewRenderUseCase(engine, minioRepo, resultProducer, a.cfg.Blender.AllowedDevices, a.cfg.Blender.Threads)
	metadataUseCase := usecase.NewMetadataUseCase(engine, minioRepo, resultProducer)

	a.consumer = kafka.NewConsumer(a.cfg.Kafka, renderUseCase, metadataUseCase, a.log, a.cfg.Blender.AllowedDevices)

	// Start metrics and health server
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	p := prometheus.NewPrometheus("worker_service")
	p.Use(r)
	r.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	a.httpSrv = &http.Server{
		Addr:    ":8081",
		Handler: r,
	}

	go func() {
		if err := a.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.log.Error("metrics server failed", slog.Any("error", err))
		}
	}()

	// Start consumer
	go func() {
		if err := a.consumer.Run(ctx); err != nil {
			a.log.Error("kafka consumer failed", slog.Any("error", err))
		}
	}()

	a.log.Info("worker service is running")

	<-ctx.Done()
	return a.Shutdown()
}

func (a *App) Shutdown() error {
	a.log.Info("shutting down worker-service")
	
	if a.consumer != nil {
		a.consumer.Close()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if a.httpSrv != nil {
		a.httpSrv.Shutdown(shutdownCtx)
	}

	return nil
}

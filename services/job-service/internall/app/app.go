package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"kseilons/job-service/internall/config"
	"kseilons/job-service/internall/producer"
	"kseilons/job-service/internall/repository/minio"
	"kseilons/job-service/internall/repository/postgres"
	transport_http "kseilons/job-service/internall/transport/http"
	transport_kafka "kseilons/job-service/internall/transport/kafka"
	"kseilons/job-service/internall/usecase"
	"kseilons/job-service/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	prometheus "github.com/zsais/go-gin-prometheus"
)

type Infrastructure struct {
	Pool     *pgxpool.Pool
	Producer *producer.KafkaProducer
	Hub      *usecase.Hub
}

type Repositories struct {
	Job   *postgres.JobRepository
	Task  *postgres.TaskRepository
	Minio *minio.MinioRepository
}

type UseCases struct {
	Job     *usecase.JobUseCase
	Cluster usecase.ClusterUseCase
}

type App struct {
	cfg      *config.Config
	log      *slog.Logger
	infra    *Infrastructure
	repos    *Repositories
	useCases *UseCases
	router   *gin.Engine
	httpSrv  *http.Server
}

func New(cfg *config.Config, log *slog.Logger) *App {
	return &App{
		cfg: cfg,
		log: log,
	}
}

func (a *App) Init(ctx context.Context) error {
	if err := a.initInfrastructure(ctx); err != nil {
		return err
	}
	if err := a.initRepositories(); err != nil {
		return err
	}
	a.initUseCases()
	a.initHandlers()
	return nil
}

func (a *App) initInfrastructure(ctx context.Context) error {
	// Database
	pool, err := pgxpool.New(ctx, a.cfg.Database.URL())
	if err != nil {
		return fmt.Errorf("failed to connect to postgres: %w", err)
	}

	// Kafka Producer
	prod := producer.NewKafkaProducer(a.cfg.Kafka.Brokers)

	// WebSocket Hub
	hub := usecase.NewHub()

	a.infra = &Infrastructure{
		Pool:     pool,
		Producer: prod,
		Hub:      hub,
	}
	return nil
}

func (a *App) initRepositories() error {
	minioRepo, err := minio.NewMinioRepository(a.cfg.Minio)
	if err != nil {
		return fmt.Errorf("failed to connect to minio: %w", err)
	}

	a.repos = &Repositories{
		Job:   postgres.NewJobRepository(a.infra.Pool),
		Task:  postgres.NewTaskRepository(a.infra.Pool),
		Minio: minioRepo,
	}
	return nil
}

func (a *App) initUseCases() {
	a.useCases = &UseCases{
		Job:     usecase.NewJobUseCase(a.repos.Job, a.repos.Task, a.repos.Minio, a.infra.Producer, a.cfg.Kafka, a.infra.Hub),
		Cluster: usecase.NewClusterUseCase(a.cfg.Http),
	}
}

func (a *App) initHandlers() {
	gin.SetMode(gin.ReleaseMode)
	if a.cfg.Env == "local" {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.Default()
	p := prometheus.NewPrometheus("job_service")
	p.Use(router)
	router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Инициализируем хендлеры
	jobHandler := transport_http.NewHandler(a.useCases.Job, a.infra.Hub)
	clusterHandler := transport_http.NewClusterHandler(a.useCases.Cluster)

	// Публичные маршруты (WebSocket)
	jobHandler.RegisterWS(router)

	// Маршруты с аутентификацией
	authGroup := router.Group("/")
	authGroup.Use(transport_http.AuthMiddleware())

	clusterHandler.Register(authGroup)
	jobHandler.Register(authGroup)

	a.router = router
	a.httpSrv = &http.Server{
		Addr:    a.cfg.Http.Address,
		Handler: router,
	}
}

func (a *App) Run(ctx context.Context) error {
	// Start background Hub
	go a.infra.Hub.Run()

	// Kafka Result Consumer
	consumer := transport_kafka.NewConsumer(
		a.cfg.Kafka.Brokers,
		a.cfg.Kafka.ResultTopic,
		a.cfg.Kafka.ResultGroupID,
		a.useCases.Job,
		a.log,
	)

	go func() {
		if err := consumer.Run(ctx); err != nil {
			a.log.Error("result consumer error", logger.Err(err))
		}
	}()

	// Start HTTP Server
	go func() {
		a.log.Info("http server starting", slog.String("address", a.cfg.Http.Address))
		if err := a.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.log.Error("http server failed", logger.Err(err))
		}
	}()

	<-ctx.Done()

	return a.Shutdown()
}

func (a *App) Shutdown() error {
	a.log.Info("shutting down job-service gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	// Clean resources
	if a.infra.Producer != nil {
		a.infra.Producer.Close()
	}
	if a.infra.Pool != nil {
		a.infra.Pool.Close()
	}

	if err := a.httpSrv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("forced server shutdown: %w", err)
	}

	a.log.Info("job-service exited")
	return nil
}

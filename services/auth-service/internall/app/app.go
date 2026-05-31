package app

import (
	"context"
	"fmt"
	"kseilons/auth-service/internall/config"
	transport_http "kseilons/auth-service/internall/transport/http"
	"kseilons/auth-service/internall/usecase"
	"log/slog"
	"net/http"
	"time"

	"github.com/Nerzal/gocloak/v13"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	prometheus "github.com/zsais/go-gin-prometheus"
)

type App struct {
	cfg     *config.Config
	log     *slog.Logger
	httpSrv *http.Server
}

func New(cfg *config.Config, log *slog.Logger) *App {
	return &App{
		cfg: cfg,
		log: log,
	}
}

func (a *App) Run(ctx context.Context) error {
	keycloakClient := gocloak.NewClient(a.cfg.Keycloak.BaseURL)
	authUseCase := usecase.NewAuthUseCase(keycloakClient, a.cfg.Keycloak)

	handler := transport_http.NewHandler(authUseCase, a.log)

	if a.cfg.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.Default()

	pprof.Register(router)
	p := prometheus.NewPrometheus("auth_service")
	p.Use(router)

	handler.Register(router)

	serverAddr := fmt.Sprintf(":%d", a.cfg.HTTP.Port)
	a.httpSrv = &http.Server{
		Addr:    serverAddr,
		Handler: router,
	}

	a.log.Info("http server starting", slog.String("addr", serverAddr))

	go func() {
		if err := a.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.log.Error("failed to start server", slog.Any("error", err))
		}
	}()

	<-ctx.Done()
	return a.Shutdown()
}

func (a *App) Shutdown() error {
	a.log.Info("shutting down auth-service")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return a.httpSrv.Shutdown(shutdownCtx)
}

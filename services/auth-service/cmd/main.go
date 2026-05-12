package main

import (
	"context"
	"fmt"
	"kseilons/auth-service/internall/app"
	"kseilons/auth-service/internall/config"
	"kseilons/auth-service/internall/transport/http"
	"kseilons/auth-service/internall/usecase"
	"kseilons/auth-service/pkg/logger"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Nerzal/gocloak/v13"
	"github.com/gin-gonic/gin"
	prometheus "github.com/zsais/go-gin-prometheus"
)

func main() {
	cfg := config.MustLoad()

	log := logger.SetupLogger(cfg.Env)
	log.Info("starting auth-service", slog.String("env", cfg.Env))

	application := app.New(cfg, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := application.Run(ctx); err != nil {
			log.Error("app exited with error", slog.Any("error", err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	<-stop
	cancel()
	log.Info("auth-service stopped")
}

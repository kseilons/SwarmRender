package main

import (
	"fmt"
	"kseilons/auth-service/internall/config"
	"kseilons/auth-service/internall/transport/http"
	"kseilons/auth-service/internall/usecase"
	"kseilons/auth-service/pkg/logger"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.MustLoad()

	log := logger.SetupLogger(cfg.Env)
	log.Info("starting auth-service", slog.String("env", cfg.Env))

	authUseCase := usecase.NewAuthUseCase(cfg.Keycloak)

	handler := http.NewHandler(authUseCase, log)

	if cfg.Env == logger.EnvProd {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.Default()

	handler.Register(router)

	serverAddr := fmt.Sprintf(":%d", cfg.HTTP.Port)
	log.Info("http server starting", slog.String("addr", serverAddr))

	go func() {
		if err := router.Run(serverAddr); err != nil {
			log.Error("failed to start server", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	<-stop

	log.Info("shutting down auth-service")
}

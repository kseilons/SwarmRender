package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kseilons/worker-service/internall/app"
	"kseilons/worker-service/internall/config"
	"kseilons/worker-service/pkg/logger"
)

func main() {
	cfg := config.MustLoad()

	log := logger.SetupLogger(cfg.Env)
	log.Info("starting worker-service", "env", cfg.Env)

	application := app.New(cfg, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := application.Run(ctx); err != nil {
			log.Error("app exited with error", "error", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	<-stop
	cancel()

	// Wait a bit for graceful shutdown
	time.Sleep(1 * time.Second)
	log.Info("worker-service stopped")
}

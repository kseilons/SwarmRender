package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"kseilons/job-service/internall/app"
	"kseilons/job-service/internall/config"
	"kseilons/job-service/pkg/logger"
)

func main() {
	// 1. Load configuration
	cfg := config.MustLoad()

	// 2. Setup context for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 3. Setup logger
	log := logger.SetupLogger(cfg.Env)
	slog.SetDefault(log)

	// 4. Initialize and Run Application
	application := app.New(cfg, log)
	
	if err := application.Init(ctx); err != nil {
		log.Error("failed to initialize application", logger.Err(err))
		os.Exit(1)
	}

	if err := application.Run(ctx); err != nil {
		log.Error("application exited with error", logger.Err(err))
		os.Exit(1)
	}
}

package main

import (
	"fmt"
	"kseilons/api-gateway/internall/config"
	transport_http "kseilons/api-gateway/internall/transport/http"
	"kseilons/api-gateway/pkg/logger"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.MustLoad()

	log := logger.SetupLogger(cfg.Env)
	log.Info("starting api-gateway", slog.String("env", cfg.Env))

	authMiddleware := transport_http.NewAuthMiddleware(log, cfg.Keycloak.JWKSURL)

	proxyHandler, err := transport_http.NewProxyHandler(log, cfg.Services.AuthService, cfg.Services.JobService)
	if err != nil {
		log.Error("failed to create proxy handler", slog.Any("error", err))
		os.Exit(1)
	}

	if cfg.Env == logger.EnvProd {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.Default()

	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	rAuth := router.Group("/auth")
	{
		rAuth.Any("/*path", proxyHandler.ProxyAuth())
	}

	rJobs := router.Group("/")
	rJobs.Use(authMiddleware.Authenticate())
	{
		rJobs.Any("/jobs/*path", proxyHandler.ProxyJob())
		rJobs.Any("/ws/*path", proxyHandler.ProxyJob())
	}

	serverAddr := fmt.Sprintf(":%d", cfg.HTTP.Port)
	log.Info("api-gateway server starting", slog.String("addr", serverAddr))

	go func() {
		if err := router.Run(serverAddr); err != nil {
			log.Error("failed to start server", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	<-stop

	log.Info("shutting down api-gateway")
}

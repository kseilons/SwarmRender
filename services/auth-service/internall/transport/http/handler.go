package http

import (
	"context"
	"kseilons/auth-service/internall/domain"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:generate mockery --name AuthUseCase --output ./mocks --case underscore
type AuthUseCase interface {
	Login(ctx context.Context, username, password string) (*domain.TokenResponse, error)
	Register(ctx context.Context, req domain.RegisterRequest) error
	RefreshToken(ctx context.Context, refreshToken string) (*domain.TokenResponse, error)
}

type Handler struct {
	useCase AuthUseCase
	log     *slog.Logger
}

func NewHandler(useCase AuthUseCase, log *slog.Logger) *Handler {
	return &Handler{
		useCase: useCase,
		log:     log,
	}
}

func (h *Handler) Register(r *gin.Engine) {
	r.GET("/health", h.Health)

	auth := r.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/register", h.RegisterUser)
		auth.POST("/refresh", h.Refresh)
	}
}

func (h *Handler) Health(c *gin.Context) {
	c.Status(http.StatusOK)
}

func (h *Handler) Login(c *gin.Context) {
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Error("failed to bind login request", slog.Any("error", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.useCase.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		h.log.Error("failed to login", slog.String("username", req.Username), slog.Any("error", err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	c.JSON(http.StatusOK, token)
}

func (h *Handler) RegisterUser(c *gin.Context) {
	var req domain.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Error("failed to bind register request", slog.Any("error", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.useCase.Register(c.Request.Context(), req)
	if err != nil {
		h.log.Error("failed to register user", slog.String("username", req.Username), slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "user created successfully"})
}

func (h *Handler) Refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Error("failed to bind refresh request", slog.Any("error", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.useCase.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		h.log.Error("failed to refresh token", slog.Any("error", err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	c.JSON(http.StatusOK, token)
}

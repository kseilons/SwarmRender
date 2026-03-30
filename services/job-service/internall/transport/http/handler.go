package http

import (
	"context"
	"net/http"
	"log/slog"
	"kseilons/job-service/internall/domain"

	"github.com/gin-gonic/gin"
)

type JobUseCase interface {
	CreateJob(ctx context.Context, job domain.Job, logger *slog.Logger) error
	UpdateJobStatus(ctx context.Context, id string, status domain.Status, logger *slog.Logger) error
}

type Handler struct {
	useCase JobUseCase
}

func NewHandler(useCase JobUseCase) *Handler {
	return &Handler{useCase: useCase}
}

func (h *Handler) Register(r *gin.Engine) {
	r.POST("/jobs", h.CreateJob)
}

func (h *Handler) CreateJob(c *gin.Context) {
	var job domain.Job
	if err := c.ShouldBindJSON(&job); err != nil {
		slog.Error("failed to bind job json", slog.Any("error", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Enrich logger with job details for this request
	logger := slog.Default().With(job.LogAttrs()...)

	logger.Info("received job creation request")

	if err := h.useCase.CreateJob(c.Request.Context(), job, logger); err != nil {
		logger.Error("failed to create job", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Info("job creation request handled successfully")
	c.JSON(http.StatusAccepted, job)
}

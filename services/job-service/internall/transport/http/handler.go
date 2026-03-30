package http

import (
	"context"
	"net/http"
	"log/slog"
	"kseilons/job-service/internall/domain"

	"github.com/gin-gonic/gin"
)

type JobUseCase interface {
	CreateJob(ctx context.Context, req domain.CreateJobRequest, logger *slog.Logger) (domain.Job, error)
	GetJob(ctx context.Context, id string) (domain.JobResponse, error)
	ListJobs(ctx context.Context) ([]domain.Job, error)
}

type Handler struct {
	useCase JobUseCase
}

func NewHandler(useCase JobUseCase) *Handler {
	return &Handler{useCase: useCase}
}

func (h *Handler) Register(r *gin.Engine) {
	r.POST("/jobs", h.CreateJob)
	r.GET("/jobs", h.ListJobs)
	r.GET("/jobs/:id", h.GetJob)
}

func (h *Handler) CreateJob(c *gin.Context) {
	var req domain.CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Error("failed to bind job json", slog.Any("error", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logger := slog.Default().With(slog.String("job_name", req.Name))
	logger.Info("received job creation request")

	job, err := h.useCase.CreateJob(c.Request.Context(), req, logger)
	if err != nil {
		logger.Error("failed to create job", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, job)
}

func (h *Handler) GetJob(c *gin.Context) {
	id := c.Param("id")
	job, err := h.useCase.GetJob(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	c.JSON(http.StatusOK, job)
}

func (h *Handler) ListJobs(c *gin.Context) {
	jobs, err := h.useCase.ListJobs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, jobs)
}

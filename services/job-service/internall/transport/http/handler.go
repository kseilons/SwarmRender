package http

import (
	"context"
	"kseilons/job-service/internall/domain"
	"kseilons/job-service/internall/usecase"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type JobUseCase interface {
	CreateJob(ctx context.Context, req domain.CreateJobRequest, logger *slog.Logger) (domain.Job, error)
	GetJob(ctx context.Context, id string) (domain.JobResponse, error)
	ListJobs(ctx context.Context) ([]domain.Job, error)
	GenerateUploadURL(ctx context.Context, filename string, projectID string) (string, string, error)
	RetryJob(ctx context.Context, id string, logger *slog.Logger) error
}

type Handler struct {
	useCase JobUseCase
	hub     *usecase.Hub
}

func NewHandler(useCase JobUseCase, hub *usecase.Hub) *Handler {
	return &Handler{
		useCase: useCase,
		hub:     hub,
	}
}

func (h *Handler) Register(r *gin.Engine) {
	r.POST("/jobs", h.CreateJob)
	r.GET("/jobs", h.ListJobs)
	r.GET("/jobs/:id", h.GetJob)
	r.POST("/jobs/:id/retry", h.RetryJob)
	r.GET("/jobs/upload-url", h.GenerateUploadURL)
	r.GET("/ws", h.HandleWS)
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
		slog.Error("failed to get job", slog.String("job_id", id), slog.Any("error", err))
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	c.JSON(http.StatusOK, job)
}

func (h *Handler) ListJobs(c *gin.Context) {
	jobs, err := h.useCase.ListJobs(c.Request.Context())
	if err != nil {
		slog.Error("failed to list jobs", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, jobs)
}

func (h *Handler) GenerateUploadURL(c *gin.Context) {
	filename := c.Query("filename")
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "filename query param is required"})
		return
	}
	projectID := c.Query("project_id")

	url, s3Path, err := h.useCase.GenerateUploadURL(c.Request.Context(), filename, projectID)
	if err != nil {
		slog.Error("failed to generate upload url", slog.String("filename", filename), slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"upload_url": url,
		"s3_path":    s3Path,
	})
}

func (h *Handler) RetryJob(c *gin.Context) {
	id := c.Param("id")
	logger := slog.Default().With(slog.String("job_id", id))
	logger.Info("received job retry request")

	err := h.useCase.RetryJob(c.Request.Context(), id, logger)
	if err != nil {
		logger.Error("failed to retry job", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "retry initiated"})
}

func (h *Handler) HandleWS(c *gin.Context) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all for now
		},
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Error("failed to upgrade to websocket", slog.Any("error", err))
		return
	}

	client := &usecase.Client{Hub: h.hub, Conn: conn, Send: make(chan []byte, 256)}
	h.hub.RegisterClient(client)

	go client.WritePump()
	go client.ReadPump()
}

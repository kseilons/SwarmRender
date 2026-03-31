package http

import (
	"context"
	"net/http"
	"kseilons/job-service/internall/domain"
	"github.com/gin-gonic/gin"
)

type ClusterUseCase interface {
	GetStats(ctx context.Context) (domain.ClusterStats, error)
}

type ClusterHandler struct {
	uc ClusterUseCase
}

func NewClusterHandler(uc ClusterUseCase) *ClusterHandler {
	return &ClusterHandler{
		uc: uc,
	}
}

func (h *ClusterHandler) Register(r *gin.Engine) {
	r.GET("/stats", h.GetStats)
}

func (h *ClusterHandler) GetStats(c *gin.Context) {
	stats, err := h.uc.GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

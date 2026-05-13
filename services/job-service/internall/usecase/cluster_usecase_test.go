package usecase_test

import (
	"context"
	"testing"

	"kseilons/job-service/internall/config"
	"kseilons/job-service/internall/usecase"

	"github.com/stretchr/testify/assert"
)

func TestClusterUseCase_GetStats(t *testing.T) {
	cfg := config.HttpServerConfig{
		MaxWorkers: 10,
	}
	uc := usecase.NewClusterUseCase(cfg)
	ctx := context.Background()

	stats, err := uc.GetStats(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 10, stats.TotalWorkers)
}

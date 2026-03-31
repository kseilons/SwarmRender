package usecase

import (
	"context"
	"kseilons/job-service/internall/config"
	"kseilons/job-service/internall/domain"
)

type ClusterUseCase interface {
	GetStats(ctx context.Context) (domain.ClusterStats, error)
}

type clusterUseCase struct {
	cfg config.HttpServerConfig
}

func NewClusterUseCase(cfg config.HttpServerConfig) ClusterUseCase {
	return &clusterUseCase{
		cfg: cfg,
	}
}

func (uc *clusterUseCase) GetStats(ctx context.Context) (domain.ClusterStats, error) {
	return domain.ClusterStats{
		TotalWorkers: uc.cfg.MaxWorkers,
	}, nil
}

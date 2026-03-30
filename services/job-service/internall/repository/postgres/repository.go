package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"kseilons/job-service/internall/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateJob(ctx context.Context, job domain.Job) error { return nil }
func (r *Repository) GetJob(ctx context.Context, id string) (domain.Job, error) { return domain.Job{}, nil }
func (r *Repository) UpdateJobStatus(ctx context.Context, id string, status domain.Status) error { return nil }

package postgres

import (
	"context"
	"fmt"
	"kseilons/job-service/internall/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type JobRepository struct {
	pool *pgxpool.Pool
}

func NewJobRepository(pool *pgxpool.Pool) *JobRepository {
	return &JobRepository{pool: pool}
}

func (r *JobRepository) CreateJob(ctx context.Context, job domain.Job) error {
	query := `
		INSERT INTO jobs (id, name, s3_path, frame_start, frame_end, worker_count, status, result_path, options, preview_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.pool.Exec(ctx, query,
		job.ID, job.Name, job.S3Path, job.FrameStart, job.FrameEnd,
		job.WorkerCount, job.Status, job.ResultPath, job.Options, job.PreviewURL, job.CreatedAt, job.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert job: %w", err)
	}
	return nil
}

func (r *JobRepository) GetJob(ctx context.Context, id string) (domain.Job, error) {
	query := `
		SELECT id, name, s3_path, frame_start, frame_end, worker_count, status, result_path, options, preview_url, created_at, updated_at
		FROM jobs WHERE id = $1
	`
	var job domain.Job
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&job.ID, &job.Name, &job.S3Path, &job.FrameStart, &job.FrameEnd,
		&job.WorkerCount, &job.Status, &job.ResultPath, &job.Options, &job.PreviewURL, &job.CreatedAt, &job.UpdatedAt,
	)
	if err != nil {
		return domain.Job{}, fmt.Errorf("failed to get job: %w", err)
	}
	return job, nil
}

func (r *JobRepository) UpdateJobStatus(ctx context.Context, id string, status domain.Status) error {
	query := `UPDATE jobs SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, string(status), id)
	return err
}

func (r *JobRepository) UpdateJobResult(ctx context.Context, id string, status domain.Status, resultPath *string) error {
	query := `UPDATE jobs SET status = $1, result_path = $2, updated_at = NOW() WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, string(status), resultPath, id)
	return err
}

func (r *JobRepository) UpdateJobMetadata(ctx context.Context, id string, frameStart, frameEnd int) error {
	query := `UPDATE jobs SET frame_start = $1, frame_end = $2, updated_at = NOW() WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, frameStart, frameEnd, id)
	return err
}

func (r *JobRepository) ListJobs(ctx context.Context) ([]domain.Job, error) {
	query := `SELECT id, name, s3_path, frame_start, frame_end, worker_count, status, result_path, preview_url, created_at FROM jobs ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []domain.Job
	for rows.Next() {
		var j domain.Job
		err := rows.Scan(&j.ID, &j.Name, &j.S3Path, &j.FrameStart, &j.FrameEnd, &j.WorkerCount, &j.Status, &j.ResultPath, &j.PreviewURL, &j.CreatedAt)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

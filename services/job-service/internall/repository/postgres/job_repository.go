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
		INSERT INTO jobs (id, name, main_file_url, frame_start, frame_end, worker_count, status, result_url, options, preview_url, completed_frames, owner_id, created_at, updated_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`
	_, err := r.pool.Exec(ctx, query,
		job.ID, job.Name, job.MainFileURL, job.FrameStart, job.FrameEnd,
		job.WorkerCount, job.Status, job.ResultURL, job.Options, job.PreviewURL, job.CompletedFrames, job.OwnerID, job.CreatedAt, job.UpdatedAt, job.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert job: %w", err)
	}
	return nil
}

func (r *JobRepository) GetJob(ctx context.Context, id string, ownerID string) (domain.Job, error) {
	query := `
		SELECT id, name, main_file_url, frame_start, frame_end, worker_count, status, result_url, options, preview_url, completed_frames, owner_id, created_at, updated_at, completed_at
		FROM jobs WHERE id = $1
	`
	args := []interface{}{id}

	if ownerID != "" {
		query += " AND owner_id = $2"
		args = append(args, ownerID)
	}

	var job domain.Job
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&job.ID, &job.Name, &job.MainFileURL, &job.FrameStart, &job.FrameEnd,
		&job.WorkerCount, &job.Status, &job.ResultURL, &job.Options, &job.PreviewURL, &job.CompletedFrames, &job.OwnerID, &job.CreatedAt, &job.UpdatedAt, &job.CompletedAt,
	)
	if err != nil {
		return domain.Job{}, fmt.Errorf("failed to get job: %w", err)
	}
	return job, nil
}

func (r *JobRepository) UpdateJobStatus(ctx context.Context, id string, status domain.Status) error {
	query := `
		UPDATE jobs 
		SET status = $1, 
		    updated_at = NOW(),
		    completed_at = CASE WHEN $1 = 'COMPLETED'::job_status THEN NOW() ELSE completed_at END
		WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, string(status), id)
	return err
}

func (r *JobRepository) UpdateJobResult(ctx context.Context, id string, status domain.Status, resultURL *string) error {
	query := `
		UPDATE jobs 
		SET status = $1, 
		    result_url = $2, 
		    updated_at = NOW(),
		    completed_at = CASE WHEN $1 = 'COMPLETED'::job_status THEN NOW() ELSE completed_at END
		WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, string(status), resultURL, id)
	return err
}

func (r *JobRepository) UpdateJobMetadata(ctx context.Context, id string, frameStart, frameEnd int) error {
	query := `UPDATE jobs SET frame_start = $1, frame_end = $2, updated_at = NOW() WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, frameStart, frameEnd, id)
	return err
}

func (r *JobRepository) UpdateJobProgress(ctx context.Context, id string, completedFrames int) error {
	query := `UPDATE jobs SET completed_frames = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, completedFrames, id)
	return err
}

func (r *JobRepository) UpdateJobPreview(ctx context.Context, id string, previewPath string) error {
	query := `UPDATE jobs SET preview_url = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, previewPath, id)
	return err
}

func (r *JobRepository) ListJobs(ctx context.Context, ownerID string) ([]domain.Job, error) {
	query := `SELECT id, name, main_file_url, frame_start, frame_end, worker_count, status, result_url, options, preview_url, completed_frames, owner_id, created_at, updated_at, completed_at FROM jobs`
	args := []interface{}{}

	if ownerID != "" {
		query += " WHERE owner_id = $1"
		args = append(args, ownerID)
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []domain.Job
	for rows.Next() {
		var j domain.Job
		err := rows.Scan(
			&j.ID, &j.Name, &j.MainFileURL, &j.FrameStart, &j.FrameEnd,
			&j.WorkerCount, &j.Status, &j.ResultURL, &j.Options,
			&j.PreviewURL, &j.CompletedFrames, &j.OwnerID, &j.CreatedAt, &j.UpdatedAt, &j.CompletedAt,
		)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

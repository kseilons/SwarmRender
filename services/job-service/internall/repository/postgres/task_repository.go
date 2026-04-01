package postgres

import (
	"context"
	"fmt"
	"kseilons/job-service/internall/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TaskRepository struct {
	pool *pgxpool.Pool
}

func NewTaskRepository(pool *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{pool: pool}
}

func (r *TaskRepository) CreateTasks(ctx context.Context, tasks []domain.Task) error {
	batch := &pgx.Batch{}
	query := `
		INSERT INTO tasks (id, job_id, type, frame_start, frame_end, status, result_path, current_frame, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	for _, t := range tasks {
		batch.Queue(query, t.ID, t.JobID, string(t.Type), t.FrameStart, t.FrameEnd, t.Status, t.ResultPath, t.CurrentFrame, t.CreatedAt, t.UpdatedAt)
	}

	results := r.pool.SendBatch(ctx, batch)
	defer results.Close()

	for range tasks {
		_, err := results.Exec()
		if err != nil {
			return fmt.Errorf("failed to insert task in batch: %w", err)
		}
	}
	return nil
}

func (r *TaskRepository) GetTask(ctx context.Context, id string) (domain.Task, error) {
	query := `SELECT id, job_id, type, frame_start, frame_end, status, worker_id, result_path, retry_count, current_frame, created_at, updated_at FROM tasks WHERE id = $1`
	var t domain.Task
	err := r.pool.QueryRow(ctx, query, id).Scan(&t.ID, &t.JobID, &t.Type, &t.FrameStart, &t.FrameEnd, &t.Status, &t.WorkerID, &t.ResultPath, &t.RetryCount, &t.CurrentFrame, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return domain.Task{}, fmt.Errorf("failed to get task: %w", err)
	}
	return t, nil
}

func (r *TaskRepository) GetTasksByJobID(ctx context.Context, jobID string) ([]domain.Task, error) {
	query := `SELECT id, job_id, type, frame_start, frame_end, status, worker_id, result_path, retry_count, current_frame, created_at, updated_at FROM tasks WHERE job_id = $1`
	rows, err := r.pool.Query(ctx, query, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []domain.Task
	for rows.Next() {
		var t domain.Task
		err := rows.Scan(&t.ID, &t.JobID, &t.Type, &t.FrameStart, &t.FrameEnd, &t.Status, &t.WorkerID, &t.ResultPath, &t.RetryCount, &t.CurrentFrame, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (r *TaskRepository) UpdateTaskProgress(ctx context.Context, id string, currentFrame int) (string, error) {
	query := `UPDATE tasks SET current_frame = $1, updated_at = NOW() WHERE id = $2 RETURNING job_id`
	var jobID string
	err := r.pool.QueryRow(ctx, query, currentFrame, id).Scan(&jobID)
	return jobID, err
}

func (r *TaskRepository) UpdateTaskStatus(ctx context.Context, id string, status domain.Status, workerID *string, resultPath *string) (string, error) {
	query := `UPDATE tasks SET status = $1, worker_id = $2, result_path = $3, updated_at = NOW() WHERE id = $4 RETURNING job_id`
	var jobID string
	err := r.pool.QueryRow(ctx, query, string(status), workerID, resultPath, id).Scan(&jobID)
	return jobID, err
}

func (r *TaskRepository) GetJobTasksStatus(ctx context.Context, jobID string) (map[domain.Status]int, error) {
	query := `SELECT status, count(*) FROM tasks WHERE job_id = $1 GROUP BY status`
	rows, err := r.pool.Query(ctx, query, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[domain.Status]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats[domain.Status(status)] = count
	}
	return stats, nil
}
func (r *TaskRepository) ResetFailedTasks(ctx context.Context, jobID string) ([]domain.Task, error) {
	query := `
		UPDATE tasks 
		SET status = 'CREATED', worker_id = NULL, result_path = NULL, retry_count = retry_count + 1, updated_at = NOW() 
		WHERE job_id = $1 AND status = 'FAILED'
		RETURNING id, job_id, type, frame_start, frame_end, status, worker_id, result_path, retry_count, current_frame, created_at, updated_at
	`
	rows, err := r.pool.Query(ctx, query, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []domain.Task
	for rows.Next() {
		var t domain.Task
		err := rows.Scan(&t.ID, &t.JobID, &t.Type, &t.FrameStart, &t.FrameEnd, &t.Status, &t.WorkerID, &t.ResultPath, &t.RetryCount, &t.CurrentFrame, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

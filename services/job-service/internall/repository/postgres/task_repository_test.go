//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"kseilons/job-service/internall/config"
	"kseilons/job-service/internall/domain"
	"kseilons/job-service/internall/repository/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

func TestTaskRepository_Integration(t *testing.T) {
	cfg, err := config.Load("../../../config/config-test.yaml")
	if err != nil {
		t.Skip("skipping integration test: config-test.yaml not found")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.Database.URL())
	if err != nil {
		t.Skip("skipping integration test: failed to connect to DB")
	}
	defer pool.Close()

	jobRepo := postgres.NewJobRepository(pool)
	taskRepo := postgres.NewTaskRepository(pool)

	jobID := uuid.New().String()
	_ = jobRepo.CreateJob(ctx, domain.Job{
		ID:          jobID,
		Name:        "task-test-job",
		S3Path:      "test/path",
		WorkerCount: 1,
		Status:      domain.StatusCreated,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})

	tasks := []domain.Task{
		{
			ID:         uuid.New().String(),
			JobID:      jobID,
			FrameStart: 1,
			FrameEnd:   50,
			Status:     domain.StatusCreated,
			CreatedAt:  time.Now().Truncate(time.Second),
			UpdatedAt:  time.Now().Truncate(time.Second),
		},
		{
			ID:         uuid.New().String(),
			JobID:      jobID,
			FrameStart: 51,
			FrameEnd:   100,
			Status:     domain.StatusCreated,
			CreatedAt:  time.Now().Truncate(time.Second),
			UpdatedAt:  time.Now().Truncate(time.Second),
		},
	}

	t.Run("CreateTasks", func(t *testing.T) {
		err := taskRepo.CreateTasks(ctx, tasks)
		assert.NoError(t, err)
	})

	t.Run("GetTasksByJobID", func(t *testing.T) {
		savedTasks, err := taskRepo.GetTasksByJobID(ctx, jobID)
		assert.NoError(t, err)
		assert.Len(t, savedTasks, 2)
	})

	t.Run("UpdateTaskStatus", func(t *testing.T) {
		taskID := tasks[0].ID
		err := taskRepo.UpdateTaskStatus(ctx, taskID, domain.StatusCompleted, "worker-1", "s3://results/part1.mp4")
		assert.NoError(t, err)

		savedTasks, _ := taskRepo.GetTasksByJobID(ctx, jobID)
		for _, st := range savedTasks {
			if st.ID == taskID {
				assert.Equal(t, domain.StatusCompleted, st.Status)
				assert.Equal(t, "worker-1", st.WorkerID)
				assert.Equal(t, "s3://results/part1.mp4", st.ResultPath)
			}
		}
	})
}

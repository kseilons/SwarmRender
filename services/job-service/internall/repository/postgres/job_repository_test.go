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

func TestJobRepository_Integration(t *testing.T) {
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

	repo := postgres.NewJobRepository(pool)

	job := domain.Job{
		ID:          uuid.New().String(),
		Name:        "integration-test-job",
		S3Path:      "test/path",
		FrameStart:  1,
		FrameEnd:    100,
		WorkerCount: 5,
		Status:      domain.StatusCreated,
		CreatedAt:   time.Now().Truncate(time.Second),
		UpdatedAt:   time.Now().Truncate(time.Second),
	}

	t.Run("CreateJob", func(t *testing.T) {
		err := repo.CreateJob(ctx, job)
		assert.NoError(t, err)
	})

	t.Run("GetJob", func(t *testing.T) {
		savedJob, err := repo.GetJob(ctx, job.ID)
		assert.NoError(t, err)
		assert.Equal(t, job.ID, savedJob.ID)
		assert.Equal(t, job.Name, savedJob.Name)
	})

	t.Run("UpdateJobStatus", func(t *testing.T) {
		err := repo.UpdateJobStatus(ctx, job.ID, domain.StatusProcessing)
		assert.NoError(t, err)

		savedJob, _ := repo.GetJob(ctx, job.ID)
		assert.Equal(t, domain.StatusProcessing, savedJob.Status)
	})

	t.Run("ListJobs", func(t *testing.T) {
		jobs, err := repo.ListJobs(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, jobs)
	})
}

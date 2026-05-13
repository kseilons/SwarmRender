package postgres_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"kseilons/job-service/internall/domain"
	"kseilons/job-service/internall/repository/postgres"
	"kseilons/job-service/internall/testutil"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositories_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	env, err := testutil.SetupIntegrationEnv(ctx)
	require.NoError(t, err)
	defer env.Terminate(ctx)

	connStr, err := env.Postgres.ConnectionString(ctx)
	require.NoError(t, err)

	// Apply migrations
	applyMigrations(t, connStr)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	defer pool.Close()

	jobRepo := postgres.NewJobRepository(pool)
	taskRepo := postgres.NewTaskRepository(pool)

	jobID := uuid.New().String()
	taskID := uuid.New().String()
	now := time.Now().Truncate(time.Microsecond)
	job := domain.Job{
		ID:          jobID,
		Name:        "integration-test",
		MainFileURL: "s3://bucket/test.blend",
		FrameStart:  1,
		FrameEnd:    10,
		WorkerCount: 2,
		Status:      domain.StatusCreated,
		OwnerID:     uuid.New().String(),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	t.Run("CreateAndGetJob", func(t *testing.T) {
		err := jobRepo.CreateJob(ctx, job)
		assert.NoError(t, err)

		got, err := jobRepo.GetJob(ctx, jobID, "")
		assert.NoError(t, err)
		assert.Equal(t, job.Name, got.Name)
		assert.Equal(t, job.OwnerID, got.OwnerID)
	})

	t.Run("UpdateJobStatus", func(t *testing.T) {
		err := jobRepo.UpdateJobStatus(ctx, jobID, domain.StatusProcessing)
		assert.NoError(t, err)

		got, err := jobRepo.GetJob(ctx, jobID, "")
		assert.NoError(t, err)
		assert.Equal(t, domain.StatusProcessing, got.Status)
	})

	t.Run("CreateTasksAndGetTasks", func(t *testing.T) {
		tasks := []domain.Task{
			{ID: taskID, JobID: jobID, Type: domain.TypeRender, FrameStart: 1, FrameEnd: 5, Status: domain.StatusCreated, CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New().String(), JobID: jobID, Type: domain.TypeRender, FrameStart: 6, FrameEnd: 10, Status: domain.StatusCreated, CreatedAt: now, UpdatedAt: now},
		}
		err := taskRepo.CreateTasks(ctx, tasks)
		assert.NoError(t, err)

		gotTasks, err := taskRepo.GetTasksByJobID(ctx, jobID)
		assert.NoError(t, err)
		assert.Len(t, gotTasks, 2)
	})

	t.Run("UpdateTaskProgressAndStatus", func(t *testing.T) {
		jid, status, err := taskRepo.UpdateTaskProgress(ctx, taskID, 3)
		assert.NoError(t, err)
		assert.Equal(t, jobID, jid)
		assert.Equal(t, domain.StatusProcessing, status)

		jid2, err := taskRepo.UpdateTaskStatus(ctx, taskID, domain.StatusCompleted, sPtr("worker-1"), sPtr("s3://res"), nil)
		assert.NoError(t, err)
		assert.Equal(t, jobID, jid2)

		task, err := taskRepo.GetTask(ctx, taskID)
		assert.NoError(t, err)
		assert.Equal(t, domain.StatusCompleted, task.Status)
		if assert.NotNil(t, task.WorkerID) {
			assert.Equal(t, "worker-1", *task.WorkerID)
		}
	})

	t.Run("GetJobTasksStatus", func(t *testing.T) {
		stats, err := taskRepo.GetJobTasksStatus(ctx, jobID)
		assert.NoError(t, err)
		assert.Equal(t, 1, stats[domain.StatusCompleted])
		assert.Equal(t, 1, stats[domain.StatusCreated])
	})

	t.Run("UpdateJobMoreDetails", func(t *testing.T) {
		err := jobRepo.UpdateJobMetadata(ctx, jobID, 10, 20)
		assert.NoError(t, err)

		err = jobRepo.UpdateJobProgress(ctx, jobID, 5)
		assert.NoError(t, err)

		got, _ := jobRepo.GetJob(ctx, jobID, "")
		assert.Equal(t, 10, got.FrameStart)
		assert.Equal(t, 20, got.FrameEnd)
		assert.Equal(t, 5, got.CompletedFrames)
	})

	t.Run("ListJobs", func(t *testing.T) {
		jobs, err := jobRepo.ListJobs(ctx, job.OwnerID)
		assert.NoError(t, err)
		assert.NotEmpty(t, jobs)
	})

	t.Run("NegativeCases", func(t *testing.T) {
		// Wrong owner
		_, err := jobRepo.GetJob(ctx, jobID, "wrong-user")
		assert.Error(t, err)

		// Not found
		_, err = jobRepo.GetJob(ctx, "non-existent", "")
		assert.Error(t, err)

		_, err = taskRepo.GetTask(ctx, "non-existent")
		assert.Error(t, err)
	})

	t.Run("UpdateJobDetails", func(t *testing.T) {
		err := jobRepo.UpdateJobPreview(ctx, jobID, "s3://preview.png")
		assert.NoError(t, err)

		err = jobRepo.UpdateJobResult(ctx, jobID, domain.StatusCompleted, sPtr("s3://final.mp4"))
		assert.NoError(t, err)

		got, _ := jobRepo.GetJob(ctx, jobID, "")
		assert.Equal(t, domain.StatusCompleted, got.Status)
		if assert.NotNil(t, got.PreviewURL) {
			assert.Equal(t, "s3://preview.png", *got.PreviewURL)
		}
		if assert.NotNil(t, got.ResultURL) {
			assert.Equal(t, "s3://final.mp4", *got.ResultURL)
		}
	})
}

func applyMigrations(t *testing.T, connStr string) {
	// Simple migration applier for tests
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	defer pool.Close()

	// Find migrations dir
	wd, _ := os.Getwd()
	// migrations are in services/job-service/migrations
	// wd is services/job-service/internall/repository/postgres
	migrationsDir := filepath.Join(wd, "..", "..", "..", "migrations")

	files, err := os.ReadDir(migrationsDir)
	require.NoError(t, err)

	for _, f := range files {
		if filepath.Ext(f.Name()) == ".sql" {
			content, err := os.ReadFile(filepath.Join(migrationsDir, f.Name()))
			require.NoError(t, err)
			_, err = pool.Exec(ctx, string(content))
			require.NoError(t, err)
		}
	}
}

func sPtr(s string) *string {
	return &s
}

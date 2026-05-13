package usecase_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"kseilons/job-service/internall/config"
	"kseilons/job-service/internall/domain"
	"kseilons/job-service/internall/usecase"
	"kseilons/job-service/internall/usecase/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestJobUseCase_ListJobs_WithPreview(t *testing.T) {
	mockJobRepo := mocks.NewJobRepository(t)
	mockMinioRepo := mocks.NewMinioRepository(t)
	uc := usecase.NewJobUseCase(mockJobRepo, nil, mockMinioRepo, nil, config.KafkaConfig{}, nil)
	ctx := context.Background()
	ownerID := "user-1"

	jobs := []domain.Job{
		{ID: "job-1", Status: domain.StatusCompleted, ResultURL: sPtr("s3://res"), PreviewURL: nil},
	}

	t.Run("Success_GeneratesPreview", func(t *testing.T) {
		mockJobRepo.On("ListJobs", ctx, ownerID).Return(jobs, nil).Once()
		mockMinioRepo.On("GenerateDownloadURL", ctx, "s3://res").Return("http://download", nil).Once()

		res, err := uc.ListJobs(ctx, domain.AuthUser{ID: ownerID})
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})

	t.Run("ListError", func(t *testing.T) {
		mockJobRepo.On("ListJobs", ctx, ownerID).Return(nil, assert.AnError).Once()
		res, err := uc.ListJobs(ctx, domain.AuthUser{ID: ownerID})
		assert.Error(t, err)
		assert.Nil(t, res)
	})
}

func TestJobUseCase_CreateJob_WithFrames(t *testing.T) {
	// Setup
	mockJobRepo := mocks.NewJobRepository(t)
	mockTaskRepo := mocks.NewTaskRepository(t)
	mockProducer := mocks.NewTaskProducer(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg := config.KafkaConfig{RenderTopic: "render-tasks"}
	uc := usecase.NewJobUseCase(mockJobRepo, mockTaskRepo, nil, mockProducer, cfg, nil)

	req := domain.CreateJobRequest{
		Name:        "test-job",
		MainFileURL: "project.blend",
		FrameStart:  1,
		FrameEnd:    10,
		WorkerCount: 2,
	}

	ctx := context.Background()
	user := domain.AuthUser{ID: "user-1"}

	// Expectations
	mockJobRepo.On("CreateJob", ctx, mock.AnythingOfType("domain.Job")).Return(nil)
	mockTaskRepo.On("CreateTasks", ctx, mock.Anything).Return(nil)
	mockProducer.On("SendTask", ctx, mock.Anything, mock.Anything, "render-tasks-cpu").Return(nil).Twice()
	mockJobRepo.On("UpdateJobStatus", ctx, mock.Anything, domain.StatusProcessing).Return(nil)

	// Execute
	job, err := uc.CreateJob(ctx, req, user, logger)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "test-job", job.Name)
	assert.Equal(t, 2, job.WorkerCount)
}

func TestJobUseCase_CreateJob_WithoutFrames(t *testing.T) {
	// Setup
	mockJobRepo := mocks.NewJobRepository(t)
	mockTaskRepo := mocks.NewTaskRepository(t)
	mockProducer := mocks.NewTaskProducer(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg := config.KafkaConfig{MetadataTopic: "meta-tasks"}
	uc := usecase.NewJobUseCase(mockJobRepo, mockTaskRepo, nil, mockProducer, cfg, nil)

	req := domain.CreateJobRequest{
		Name:        "meta-job",
		MainFileURL: "project.blend",
		WorkerCount: 1,
	}

	ctx := context.Background()
	user := domain.AuthUser{ID: "user-meta"}

	// Expectations
	mockJobRepo.On("CreateJob", ctx, mock.AnythingOfType("domain.Job")).Return(nil)
	mockProducer.On("SendTask", ctx, mock.Anything, mock.MatchedBy(func(task any) bool {
		_, ok := task.(domain.MetadataTask)
		return ok
	}), "meta-tasks").Return(nil)
	mockJobRepo.On("UpdateJobStatus", ctx, mock.Anything, domain.StatusPendingMetadata).Return(nil)

	// Execute
	job, err := uc.CreateJob(ctx, req, user, logger)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, domain.StatusCreated, job.Status)
}

func TestJobUseCase_HandleMetadataResult(t *testing.T) {
	// Setup
	mockJobRepo := mocks.NewJobRepository(t)
	mockTaskRepo := mocks.NewTaskRepository(t)
	mockProducer := mocks.NewTaskProducer(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	uc := usecase.NewJobUseCase(mockJobRepo, mockTaskRepo, nil, mockProducer, config.KafkaConfig{}, nil)

	jobID := uuid.New().String()
	ctx := context.Background()

	// Expectations
	mockJobRepo.On("UpdateJobMetadata", ctx, jobID, 1, 100).Return(nil)
	mockJobRepo.On("GetJob", ctx, jobID, "").Return(domain.Job{
		ID:          jobID,
		WorkerCount: 1,
		FrameStart:  1,
		FrameEnd:    100,
		MainFileURL: "project.blend",
	}, nil)
	mockTaskRepo.On("CreateTasks", ctx, mock.Anything).Return(nil)
	mockProducer.On("SendTask", ctx, jobID, mock.Anything, mock.Anything).Return(nil)
	mockJobRepo.On("UpdateJobStatus", ctx, jobID, domain.StatusProcessing).Return(nil)

	// Execute
	err := uc.HandleMetadataResult(ctx, jobID, 1, 100, logger)

	// Assert
	assert.NoError(t, err)
}

func TestJobUseCase_CreateJob_RepoFailure(t *testing.T) {
	mockJobRepo := mocks.NewJobRepository(t)
	uc := usecase.NewJobUseCase(mockJobRepo, nil, nil, nil, config.KafkaConfig{}, nil)

	req := domain.CreateJobRequest{Name: "fail", MainFileURL: "path", WorkerCount: 1}
	ctx := context.Background()
	user := domain.AuthUser{ID: "user-fail"}

	mockJobRepo.On("CreateJob", ctx, mock.Anything).Return(assert.AnError).Once()

	_, err := uc.CreateJob(ctx, req, user, slog.Default())
	assert.Error(t, err)
}

func TestJobUseCase_GeneratePreview(t *testing.T) {
	mockJobRepo := mocks.NewJobRepository(t)
	mockMinioRepo := mocks.NewMinioRepository(t)
	uc := usecase.NewJobUseCase(mockJobRepo, nil, mockMinioRepo, nil, config.KafkaConfig{}, nil)
	ctx := context.Background()
	logger := slog.Default()

	job := domain.Job{ID: "j1", MainFileURL: "p1_sources/scene.blend", Options: domain.RenderOptions{Format: "PNG"}}
	task := domain.Task{FrameStart: 1}

	mockMinioRepo.On("CopyObject", ctx, "p1_results/0001.png", "p1_preview.png").Return(nil)
	mockJobRepo.On("UpdateJobPreview", ctx, "j1", "p1_preview.png").Return(nil)

	uc.ExportGeneratePreview(ctx, job, task, logger)
}

func TestJobUseCase_HandleTaskResult_PartialFailure(t *testing.T) {
	mockJobRepo := mocks.NewJobRepository(t)
	mockTaskRepo := mocks.NewTaskRepository(t)
	uc := usecase.NewJobUseCase(mockJobRepo, mockTaskRepo, nil, nil, config.KafkaConfig{}, nil)
	ctx := context.Background()
	logger := slog.Default()

	jobID := "j1"
	taskID := "t1"

	mockTaskRepo.On("UpdateTaskStatus", ctx, taskID, domain.StatusFailed, mock.Anything, mock.Anything, mock.Anything).Return(jobID, nil)
	mockTaskRepo.On("GetJobTasksStatus", ctx, jobID).Return(map[domain.Status]int{
		domain.StatusCompleted: 1,
		domain.StatusFailed:    1,
	}, nil)
	mockJobRepo.On("UpdateJobStatus", ctx, jobID, domain.StatusFailed).Return(nil)

	err := uc.HandleTaskResult(ctx, domain.RenderResult{TaskID: taskID, Status: domain.StatusFailed}, logger)
	assert.NoError(t, err)
}

func TestJobUseCase_GetJob(t *testing.T) {
	mockJobRepo := mocks.NewJobRepository(t)
	mockTaskRepo := mocks.NewTaskRepository(t)
	uc := usecase.NewJobUseCase(mockJobRepo, mockTaskRepo, nil, nil, config.KafkaConfig{}, nil)

	jobID := "job-123"
	ctx := context.Background()
	user := domain.AuthUser{ID: "user-123"}

	t.Run("Success", func(t *testing.T) {
		mockJobRepo.On("GetJob", ctx, jobID, user.ID).Return(domain.Job{ID: jobID}, nil).Once()
		mockTaskRepo.On("GetTasksByJobID", ctx, jobID).Return([]domain.Task{{ID: "task-1"}}, nil).Once()

		resp, err := uc.GetJob(ctx, jobID, user)
		assert.NoError(t, err)
		assert.Equal(t, jobID, resp.Job.ID)
		assert.Len(t, resp.Tasks, 1)
	})

	t.Run("WithURLs", func(t *testing.T) {
		mockMinioRepo := mocks.NewMinioRepository(t)
		ucWithMinio := usecase.NewJobUseCase(mockJobRepo, mockTaskRepo, mockMinioRepo, nil, config.KafkaConfig{}, nil)

		preview := "preview.png"
		result := "result.zip"
		job := domain.Job{ID: jobID, PreviewURL: &preview, ResultURL: &result}

		mockJobRepo.On("GetJob", ctx, jobID, user.ID).Return(job, nil).Once()
		mockTaskRepo.On("GetTasksByJobID", ctx, jobID).Return([]domain.Task{}, nil).Once()

		mockMinioRepo.On("GenerateDownloadURL", ctx, preview).Return("http://s3/preview", nil).Once()
		mockMinioRepo.On("GenerateDownloadURL", ctx, result).Return("http://s3/result", nil).Once()

		resp, err := ucWithMinio.GetJob(ctx, jobID, user)
		assert.NoError(t, err)
		assert.Equal(t, "http://s3/preview", *resp.Job.PreviewURL)
		assert.Equal(t, "http://s3/result", *resp.Job.ResultURL)
	})

	t.Run("JobNotFound", func(t *testing.T) {
		mockJobRepo.On("GetJob", ctx, "missing", user.ID).Return(domain.Job{}, assert.AnError).Once()
		_, err := uc.GetJob(ctx, "missing", user)
		assert.Error(t, err)
	})

	t.Run("TasksError", func(t *testing.T) {
		mockJobRepo.On("GetJob", ctx, jobID, user.ID).Return(domain.Job{ID: jobID}, nil).Once()
		mockTaskRepo.On("GetTasksByJobID", ctx, jobID).Return(nil, assert.AnError).Once()

		_, err := uc.GetJob(ctx, jobID, user)
		assert.Error(t, err)
	})
}

func TestJobUseCase_ListJobs(t *testing.T) {
	mockJobRepo := mocks.NewJobRepository(t)
	uc := usecase.NewJobUseCase(mockJobRepo, nil, nil, nil, config.KafkaConfig{}, nil)

	ctx := context.Background()
	user := domain.AuthUser{ID: "user-list"}

	t.Run("Success", func(t *testing.T) {
		mockJobRepo.On("ListJobs", ctx, user.ID).Return([]domain.Job{{ID: "1"}, {ID: "2"}}, nil).Once()
		jobs, err := uc.ListJobs(ctx, user)
		assert.NoError(t, err)
		assert.Len(t, jobs, 2)
	})

	t.Run("WithURLs", func(t *testing.T) {
		mockMinioRepo := mocks.NewMinioRepository(t)
		ucWithMinio := usecase.NewJobUseCase(mockJobRepo, nil, mockMinioRepo, nil, config.KafkaConfig{}, nil)

		preview := "p.png"
		job := domain.Job{ID: "1", PreviewURL: &preview}

		mockJobRepo.On("ListJobs", ctx, user.ID).Return([]domain.Job{job}, nil).Once()
		mockMinioRepo.On("GenerateDownloadURL", ctx, preview).Return("http://s3/p", nil).Once()

		jobs, err := ucWithMinio.ListJobs(ctx, user)
		assert.NoError(t, err)
		assert.Equal(t, "http://s3/p", *jobs[0].PreviewURL)
	})

	t.Run("Error", func(t *testing.T) {
		mockJobRepo.On("ListJobs", ctx, user.ID).Return(nil, assert.AnError).Once()
		_, err := uc.ListJobs(ctx, user)
		assert.Error(t, err)
	})
}

func TestJobUseCase_HandleTaskResult(t *testing.T) {
	result := domain.RenderResult{
		TaskID:    "task-1",
		Type:      domain.TypeRender,
		Status:    domain.StatusCompleted,
		WorkerID:  sPtr("worker-1"),
		ResultURL: sPtr("s3://output"),
	}
	jobID := "job-1"
	ctx := context.Background()

	t.Run("TaskCompleted_NotAllFinished", func(t *testing.T) {
		mockTaskRepo := mocks.NewTaskRepository(t)
		mockJobRepo := mocks.NewJobRepository(t)
		uc := usecase.NewJobUseCase(mockJobRepo, mockTaskRepo, nil, nil, config.KafkaConfig{}, nil)

		mockTaskRepo.On("UpdateTaskStatus", ctx, result.TaskID, result.Status, result.WorkerID, result.ResultURL, mock.Anything).Return(jobID, nil).Once()
		mockTaskRepo.On("GetTask", ctx, result.TaskID).Return(domain.Task{ID: result.TaskID, FrameEnd: 5}, nil).Once()
		mockTaskRepo.On("UpdateTaskProgress", ctx, result.TaskID, 5).Return(jobID, domain.StatusProcessing, nil).Once()
		mockTaskRepo.On("GetTasksByJobID", ctx, jobID).Return([]domain.Task{
			{ID: result.TaskID, Status: domain.StatusCompleted, FrameStart: 1, FrameEnd: 5},
			{ID: "task-2", Status: domain.StatusProcessing, FrameStart: 6, FrameEnd: 10},
		}, nil).Once()
		mockJobRepo.On("UpdateJobProgress", ctx, jobID, 5).Return(nil).Once()
		mockTaskRepo.On("GetJobTasksStatus", ctx, jobID).Return(map[domain.Status]int{
			domain.StatusCompleted:  1,
			domain.StatusProcessing: 1,
		}, nil).Once()
		mockJobRepo.On("GetJob", ctx, jobID, "").Return(domain.Job{ID: jobID, PreviewURL: sPtr("path")}, nil).Once()

		err := uc.HandleTaskResult(ctx, result, slog.Default())
		assert.NoError(t, err)
	})

	t.Run("AllTasksCompleted_TriggersStitching", func(t *testing.T) {
		mockTaskRepo := mocks.NewTaskRepository(t)
		mockJobRepo := mocks.NewJobRepository(t)
		mockProducer := mocks.NewTaskProducer(t)
		uc := usecase.NewJobUseCase(mockJobRepo, mockTaskRepo, nil, mockProducer, config.KafkaConfig{StitchingTopic: "stitching-tasks"}, nil)

		mockTaskRepo.On("UpdateTaskStatus", ctx, result.TaskID, result.Status, result.WorkerID, result.ResultURL, mock.Anything).Return(jobID, nil).Once()
		mockTaskRepo.On("GetTask", ctx, result.TaskID).Return(domain.Task{ID: result.TaskID, FrameEnd: 10}, nil).Once()
		mockTaskRepo.On("UpdateTaskProgress", ctx, result.TaskID, 10).Return(jobID, domain.StatusProcessing, nil).Once()
		mockTaskRepo.On("GetTasksByJobID", ctx, jobID).Return([]domain.Task{
			{ID: result.TaskID, Status: domain.StatusCompleted, FrameStart: 1, FrameEnd: 10},
		}, nil).Once()
		mockJobRepo.On("UpdateJobProgress", ctx, jobID, 10).Return(nil).Once()
		mockTaskRepo.On("GetJobTasksStatus", ctx, jobID).Return(map[domain.Status]int{
			domain.StatusCompleted: 1,
		}, nil).Once()

		mockJobRepo.On("GetJob", ctx, jobID, "").Return(domain.Job{ID: jobID, FrameStart: 1, FrameEnd: 10, MainFileURL: "test.blend"}, nil).Once()
		mockJobRepo.On("UpdateJobStatus", ctx, jobID, domain.StatusStitching).Return(nil).Once()
		mockTaskRepo.On("CreateTasks", ctx, mock.Anything).Return(nil).Once()
		mockProducer.On("SendTask", ctx, mock.Anything, mock.Anything, "stitching-tasks").Return(nil).Once()

		err := uc.HandleTaskResult(ctx, result, slog.Default())
		assert.NoError(t, err)
	})

	t.Run("AllTasksFailed_TriggersFailed", func(t *testing.T) {
		mockTaskRepo := mocks.NewTaskRepository(t)
		mockJobRepo := mocks.NewJobRepository(t)
		uc := usecase.NewJobUseCase(mockJobRepo, mockTaskRepo, nil, nil, config.KafkaConfig{}, nil)

		failResult := result
		failResult.Status = domain.StatusFailed

		mockTaskRepo.On("UpdateTaskStatus", ctx, failResult.TaskID, failResult.Status, failResult.WorkerID, failResult.ResultURL, mock.Anything).Return(jobID, nil).Once()
		mockTaskRepo.On("GetJobTasksStatus", ctx, jobID).Return(map[domain.Status]int{
			domain.StatusFailed: 2,
		}, nil).Once()
		mockJobRepo.On("UpdateJobStatus", ctx, jobID, domain.StatusFailed).Return(nil).Once()

		err := uc.HandleTaskResult(ctx, failResult, slog.Default())
		assert.NoError(t, err)
	})

	t.Run("UpdateStatusError", func(t *testing.T) {
		mockTaskRepo := mocks.NewTaskRepository(t)
		uc := usecase.NewJobUseCase(nil, mockTaskRepo, nil, nil, config.KafkaConfig{}, nil)
		mockTaskRepo.On("UpdateTaskStatus", ctx, "task-1", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("", assert.AnError).Once()
		err := uc.HandleTaskResult(ctx, result, slog.Default())
		assert.Error(t, err)
	})
}

func TestJobUseCase_HandleStitchingResult(t *testing.T) {
	mockTaskRepo := mocks.NewTaskRepository(t)
	mockJobRepo := mocks.NewJobRepository(t)
	uc := usecase.NewJobUseCase(mockJobRepo, mockTaskRepo, nil, nil, config.KafkaConfig{}, nil)
	ctx := context.Background()
	jobID := "job-1"
	taskID := "stitch-1"
	result := domain.RenderResult{TaskID: taskID, Status: domain.StatusCompleted, Type: domain.TypeStitching, ResultURL: sPtr("s3://final")}

	mockTaskRepo.On("UpdateTaskStatus", ctx, taskID, domain.StatusCompleted, mock.Anything, mock.Anything, mock.Anything).Return(jobID, nil).Once()
	mockJobRepo.On("UpdateJobResult", ctx, jobID, domain.StatusCompleted, result.ResultURL).Return(nil).Once()

	err := uc.HandleStitchingResult(ctx, result, slog.Default())
	assert.NoError(t, err)
}

func TestJobUseCase_HandleTaskResult_WithPreview(t *testing.T) {
	mockTaskRepo := mocks.NewTaskRepository(t)
	mockJobRepo := mocks.NewJobRepository(t)
	mockMinioRepo := mocks.NewMinioRepository(t)
	uc := usecase.NewJobUseCase(mockJobRepo, mockTaskRepo, mockMinioRepo, nil, config.KafkaConfig{}, nil)
	ctx := context.Background()
	jobID := "job-1"
	taskID := "task-1"
	result := domain.RenderResult{TaskID: taskID, Status: domain.StatusCompleted, Type: domain.TypeRender}

	mockTaskRepo.On("UpdateTaskStatus", ctx, taskID, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(jobID, nil).Once()
	mockTaskRepo.On("GetTask", ctx, taskID).Return(domain.Task{ID: taskID, FrameStart: 1, FrameEnd: 1}, nil).Twice()
	mockTaskRepo.On("UpdateTaskProgress", ctx, taskID, 1).Return(jobID, domain.StatusProcessing, nil).Once()
	mockTaskRepo.On("GetTasksByJobID", ctx, jobID).Return([]domain.Task{
		{ID: taskID, FrameStart: 1, FrameEnd: 1, CurrentFrame: 1, Status: domain.StatusCompleted},
		{ID: "task-2", FrameStart: 2, FrameEnd: 10, Status: domain.StatusProcessing}, // Keep one task pending to avoid stitching
	}, nil).Once()
	mockJobRepo.On("UpdateJobProgress", ctx, jobID, 1).Return(nil).Once()
	mockTaskRepo.On("GetJobTasksStatus", ctx, jobID).Return(map[domain.Status]int{
		domain.StatusCompleted:  1,
		domain.StatusProcessing: 1,
	}, nil).Once()
	mockJobRepo.On("GetJob", ctx, jobID, "").Return(domain.Job{ID: jobID, MainFileURL: "p_sources/t.blend", Options: domain.RenderOptions{Format: "PNG"}}, nil).Once()

	// Preview generation mocks
	mockMinioRepo.On("CopyObject", ctx, "p_results/0001.png", "p_preview.png").Return(nil).Once()
	mockJobRepo.On("UpdateJobPreview", ctx, jobID, "p_preview.png").Return(nil).Once()

	err := uc.HandleTaskResult(ctx, result, slog.Default())
	assert.NoError(t, err)
}

func TestJobUseCase_UpdateJobStatus_Error(t *testing.T) {
	mockJobRepo := mocks.NewJobRepository(t)
	uc := usecase.NewJobUseCase(mockJobRepo, nil, nil, nil, config.KafkaConfig{}, nil)
	ctx := context.Background()

	mockJobRepo.On("UpdateJobStatus", ctx, "job-1", domain.StatusProcessing).Return(assert.AnError).Once()
	err := uc.UpdateJobStatus(ctx, "job-1", domain.StatusProcessing, slog.Default())
	assert.Error(t, err)
}

func TestJobUseCase_RequestMetadata_Error(t *testing.T) {
	mockProducer := mocks.NewTaskProducer(t)
	mockJobRepo := mocks.NewJobRepository(t)
	uc := usecase.NewJobUseCase(mockJobRepo, nil, nil, mockProducer, config.KafkaConfig{MetadataTopic: "meta"}, nil)
	ctx := context.Background()

	job := domain.Job{ID: "job-1"}
	mockProducer.On("SendTask", ctx, mock.Anything, mock.Anything, "meta").Return(assert.AnError).Once()

	err := uc.RequestMetadata(ctx, job, slog.Default())
	assert.Error(t, err)
}

func TestJobUseCase_RetryJob(t *testing.T) {
	mockJobRepo := mocks.NewJobRepository(t)
	mockTaskRepo := mocks.NewTaskRepository(t)
	mockProducer := mocks.NewTaskProducer(t)
	uc := usecase.NewJobUseCase(mockJobRepo, mockTaskRepo, nil, mockProducer, config.KafkaConfig{RenderTopic: "render"}, nil)
	ctx := context.Background()
	user := domain.AuthUser{ID: "user-1"}
	jobID := "job-1"

	t.Run("Success_WithTasks", func(t *testing.T) {
		mockJobRepo.On("GetJob", ctx, jobID, user.ID).Return(domain.Job{ID: jobID, MainFileURL: "proj_sources/test.blend"}, nil).Once()
		mockTaskRepo.On("ResetFailedTasks", ctx, jobID).Return([]domain.Task{{ID: "task-1", FrameStart: 1, FrameEnd: 1}}, nil).Once()
		mockProducer.On("SendTask", ctx, jobID, mock.Anything, "render-cpu").Return(nil).Once()
		mockJobRepo.On("UpdateJobStatus", ctx, jobID, domain.StatusProcessing).Return(nil).Once()

		err := uc.RetryJob(ctx, jobID, user, slog.Default())
		assert.NoError(t, err)
	})

	t.Run("Success_NoTasks", func(t *testing.T) {
		mockJobRepo.On("GetJob", ctx, jobID, user.ID).Return(domain.Job{ID: jobID}, nil).Once()
		mockTaskRepo.On("ResetFailedTasks", ctx, jobID).Return([]domain.Task{}, nil).Once()

		err := uc.RetryJob(ctx, jobID, user, slog.Default())
		assert.NoError(t, err)
	})

	t.Run("Error_GetJob", func(t *testing.T) {
		mockJobRepo.On("GetJob", ctx, jobID, user.ID).Return(domain.Job{}, assert.AnError).Once()
		err := uc.RetryJob(ctx, jobID, user, slog.Default())
		assert.Error(t, err)
	})
}

func TestJobUseCase_GenerateUploadURL(t *testing.T) {
	mockMinioRepo := mocks.NewMinioRepository(t)
	uc := usecase.NewJobUseCase(nil, nil, mockMinioRepo, nil, config.KafkaConfig{}, nil)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mockMinioRepo.On("GenerateUploadURL", ctx, mock.Anything).Return("http://upload", nil).Once()
		url, key, err := uc.GenerateUploadURL(ctx, "test.blend", "proj-1")
		assert.NoError(t, err)
		assert.Equal(t, "http://upload", url)
		assert.Contains(t, key, "proj-1_sources/test.blend")
	})

	t.Run("Error", func(t *testing.T) {
		mockMinioRepo.On("GenerateUploadURL", ctx, mock.Anything).Return("", assert.AnError).Once()
		_, _, err := uc.GenerateUploadURL(ctx, "test.blend", "")
		assert.Error(t, err)
	})
}

func TestJobUseCase_SplitJobIntoTasks_Error(t *testing.T) {
	uc := usecase.NewJobUseCase(nil, nil, nil, nil, config.KafkaConfig{}, nil)
	ctx := context.Background()
	job := domain.Job{FrameStart: 10, FrameEnd: 1} // Invalid range

	err := uc.SplitJobIntoTasks(ctx, job, slog.Default())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid frame range")
}

func TestJobUseCase_HandleTaskProgress(t *testing.T) {
	mockTaskRepo := mocks.NewTaskRepository(t)
	mockJobRepo := mocks.NewJobRepository(t)
	uc := usecase.NewJobUseCase(mockJobRepo, mockTaskRepo, nil, nil, config.KafkaConfig{}, nil)
	ctx := context.Background()
	taskID := "task-1"
	jobID := "job-1"

	t.Run("Success", func(t *testing.T) {
		mockTaskRepo.On("UpdateTaskProgress", ctx, taskID, 5).Return(jobID, domain.StatusProcessing, nil).Once()
		mockTaskRepo.On("GetTasksByJobID", ctx, jobID).Return([]domain.Task{
			{ID: taskID, FrameStart: 1, FrameEnd: 10, CurrentFrame: 5, Status: domain.StatusProcessing},
		}, nil).Once()
		mockJobRepo.On("UpdateJobProgress", ctx, jobID, 5).Return(nil).Once()

		err := uc.HandleTaskProgress(ctx, taskID, 5, slog.Default())
		assert.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		mockTaskRepo.On("UpdateTaskProgress", ctx, taskID, 5).Return("", domain.StatusProcessing, assert.AnError).Once()
		err := uc.HandleTaskProgress(ctx, taskID, 5, slog.Default())
		assert.Error(t, err)
	})
}

func TestJobUseCase_SetKafkaConfig(t *testing.T) {
	uc := usecase.NewJobUseCase(nil, nil, nil, nil, config.KafkaConfig{}, nil)
	uc.SetKafkaConfig(config.KafkaConfig{RenderTopic: "new-topic"})
}

func TestJobUseCase_GetJob_Error(t *testing.T) {
	mockJobRepo := mocks.NewJobRepository(t)
	uc := usecase.NewJobUseCase(mockJobRepo, nil, nil, nil, config.KafkaConfig{}, nil)
	ctx := context.Background()

	mockJobRepo.On("GetJob", ctx, "job-1", "user-1").Return(domain.Job{}, assert.AnError).Once()

	res, err := uc.GetJob(ctx, "job-1", domain.AuthUser{ID: "user-1"})
	assert.Error(t, err)
	assert.Empty(t, res)
}

func TestJobUseCase_RetryJob_NotFound(t *testing.T) {
	mockJobRepo := mocks.NewJobRepository(t)
	uc := usecase.NewJobUseCase(mockJobRepo, nil, nil, nil, config.KafkaConfig{}, nil)
	ctx := context.Background()
	logger := slog.Default()

	mockJobRepo.On("GetJob", ctx, "job-1", "user-1").Return(domain.Job{}, assert.AnError).Once()

	err := uc.RetryJob(ctx, "job-1", domain.AuthUser{ID: "user-1"}, logger)
	assert.Error(t, err)
}

func sPtr(s string) *string {
	return &s
}

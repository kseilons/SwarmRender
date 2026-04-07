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

func sPtr(s string) *string {
	return &s
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
		mockJobRepo.On("UpdateJobProgress", ctx, jobID, 5).Return(nil).Once()
		mockTaskRepo.On("GetTasksByJobID", ctx, jobID).Return([]domain.Task{{ID: result.TaskID, Status: domain.StatusCompleted, FrameStart: 1, FrameEnd: 5}}, nil).Maybe()
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
		mockJobRepo.On("UpdateJobProgress", ctx, jobID, 10).Return(nil).Once()
		mockTaskRepo.On("GetTasksByJobID", ctx, jobID).Return([]domain.Task{{ID: result.TaskID, Status: domain.StatusCompleted, FrameStart: 1, FrameEnd: 10}}, nil).Maybe()
		mockTaskRepo.On("GetJobTasksStatus", ctx, jobID).Return(map[domain.Status]int{
			domain.StatusCompleted: 2,
		}, nil).Once()

		mockJobRepo.On("GetJob", ctx, jobID, "").Return(domain.Job{ID: jobID, FrameStart: 1, FrameEnd: 10, MainFileURL: "test.blend"}, nil).Once()
		mockJobRepo.On("UpdateJobStatus", ctx, jobID, domain.StatusStitching).Return(nil).Once()
		mockTaskRepo.On("CreateTasks", ctx, mock.Anything).Return(nil).Once()
		mockProducer.On("SendTask", ctx, mock.Anything, mock.Anything, "stitching-tasks").Return(nil).Once()

		err := uc.HandleTaskResult(ctx, result, slog.Default())
		assert.NoError(t, err)
	})

	t.Run("StitchingResult_FinalizesJob", func(t *testing.T) {
		mockTaskRepo := mocks.NewTaskRepository(t)
		mockJobRepo := mocks.NewJobRepository(t)
		uc := usecase.NewJobUseCase(mockJobRepo, mockTaskRepo, nil, nil, config.KafkaConfig{}, nil)

		taskID := "stitch-task-uuid"
		stitchResult := domain.RenderResult{
			TaskID:    taskID,
			Type:      domain.TypeStitching,
			Status:    domain.StatusCompleted,
			ResultURL: sPtr("s3://final/video.mp4"),
		}

		mockTaskRepo.On("UpdateTaskStatus", ctx, taskID, domain.StatusCompleted, (*string)(nil), stitchResult.ResultURL, mock.Anything).Return(jobID, nil).Once()
		mockJobRepo.On("UpdateJobResult", ctx, jobID, domain.StatusCompleted, stitchResult.ResultURL).Return(nil).Once()

		err := uc.HandleStitchingResult(ctx, stitchResult, slog.Default())
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

func TestJobUseCase_SetKafkaConfig(t *testing.T) {
	uc := usecase.NewJobUseCase(nil, nil, nil, nil, config.KafkaConfig{}, nil)
	uc.SetKafkaConfig(config.KafkaConfig{RenderTopic: "new-topic"})
}

package usecase_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"kseilons/worker-service/internall/domain"
	"kseilons/worker-service/internall/usecase"
	"kseilons/worker-service/internall/usecase/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRenderUseCase_Execute_Success(t *testing.T) {
	// Setup
	mockRenderer := mocks.NewRenderer(t)
	mockRepo := mocks.NewProjectRepository(t)
	mockProducer := mocks.NewResultProducer(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	allowedDevices := []string{"CUDA", "OPTIX"}
	uc := usecase.NewRenderUseCase(mockRenderer, mockRepo, mockProducer, allowedDevices, 4)

	task := domain.RenderTask{
		BaseTask: domain.BaseTask{
			ProjectID: "test-proj",
			TaskID:    "task-123",
		},
		BlendFile: "scene.blend",
		CommonTask: domain.CommonTask{
			StartFrame: 1,
			EndFrame:   1,
			Options: domain.RenderOptions{
				Engine:       domain.EngineCycles,
				CyclesDevice: domain.DeviceCUDA,
			},
		},
	}

	ctx := context.Background()

	// Expectations
	mockRepo.On("DownloadProject", ctx, "test-proj", mock.AnythingOfType("string")).Return(nil)
	mockRenderer.On("Render", ctx, logger, mock.MatchedBy(func(job domain.RenderJob) bool {
		return job.Options.Threads == 4
	}), mock.Anything).Return([]byte("rendered-data"), nil)
	mockRepo.On("UploadResult", ctx, "test-proj", mock.AnythingOfType("string")).Return(nil)
	mockProducer.On("ReportResult", ctx, mock.MatchedBy(func(res domain.RenderResult) bool {
		return res.TaskID == "task-123" && res.ProjectID == "test-proj" && res.Type == domain.TypeRender && res.Status == domain.StatusCompleted
	})).Return(nil)

	// Execute
	output, err := uc.Execute(ctx, logger, task)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, []byte("rendered-data"), output)
}

func TestRenderUseCase_Execute_RenderFailure(t *testing.T) {
	// Setup
	mockRenderer := mocks.NewRenderer(t)
	mockRepo := mocks.NewProjectRepository(t)
	mockProducer := mocks.NewResultProducer(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	uc := usecase.NewRenderUseCase(mockRenderer, mockRepo, mockProducer, nil, 0)
	task := domain.RenderTask{
		BaseTask: domain.BaseTask{
			ProjectID: "test-proj",
			TaskID:    "task-failed",
		},
	}
	ctx := context.Background()

	// Expectations
	mockRepo.On("DownloadProject", ctx, "test-proj", mock.Anything).Return(nil)
	mockRenderer.On("Render", ctx, logger, mock.Anything, mock.Anything).Return(nil, errors.New("blender crash"))
	mockProducer.On("ReportResult", ctx, mock.MatchedBy(func(res domain.RenderResult) bool {
		return res.TaskID == "task-failed" && res.ProjectID == "test-proj" && res.Type == domain.TypeRender && res.Status == domain.StatusFailed && res.Error == "blender crash"
	})).Return(nil)

	// Execute
	output, err := uc.Execute(ctx, logger, task)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, output)
}

func TestRenderUseCase_Execute_ResultReportFailure(t *testing.T) {
	// Setup
	mockRenderer := mocks.NewRenderer(t)
	mockRepo := mocks.NewProjectRepository(t)
	mockProducer := mocks.NewResultProducer(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	uc := usecase.NewRenderUseCase(mockRenderer, mockRepo, mockProducer, nil, 0)

	task := domain.RenderTask{
		BaseTask: domain.BaseTask{
			ProjectID: "test-proj",
			TaskID:    "task-fail-report",
		},
	}

	ctx := context.Background()

	// Expectations
	mockRepo.On("DownloadProject", ctx, "test-proj", mock.AnythingOfType("string")).Return(nil)
	mockRenderer.On("Render", ctx, logger, mock.Anything, mock.Anything).Return([]byte("ok"), nil)
	mockRepo.On("UploadResult", ctx, "test-proj", mock.AnythingOfType("string")).Return(nil)
	mockProducer.On("ReportResult", ctx, mock.Anything).Return(errors.New("kafka down"))

	// Execute
	output, err := uc.Execute(ctx, logger, task)

	// Assert
	// Execute should still succeed even if reporting fails (it just logs the error)
	assert.NoError(t, err)
	assert.Equal(t, []byte("ok"), output)
}

func TestRenderUseCase_Execute_ValidationFailure(t *testing.T) {
	// Setup
	mockRenderer := mocks.NewRenderer(t)
	mockRepo := mocks.NewProjectRepository(t)
	mockProducer := mocks.NewResultProducer(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	allowedDevices := []string{"CPU"}
	uc := usecase.NewRenderUseCase(mockRenderer, mockRepo, mockProducer, allowedDevices, 0)

	task := domain.RenderTask{
		BaseTask: domain.BaseTask{
			ProjectID: "test-proj",
		},
		CommonTask: domain.CommonTask{
			Options: domain.RenderOptions{
				Engine:       domain.EngineCycles,
				CyclesDevice: domain.DeviceCUDA, // Not allowed
			},
		},
	}

	ctx := context.Background()

	// Execute
	output, err := uc.Execute(ctx, logger, task)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
	assert.Nil(t, output)
	mockRepo.AssertNotCalled(t, "DownloadProject", mock.Anything, mock.Anything, mock.Anything)
}

func TestRenderUseCase_Execute_DownloadFailure(t *testing.T) {
	// Setup
	mockRenderer := mocks.NewRenderer(t)
	mockRepo := mocks.NewProjectRepository(t)
	mockProducer := mocks.NewResultProducer(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	uc := usecase.NewRenderUseCase(mockRenderer, mockRepo, mockProducer, nil, 0)

	task := domain.RenderTask{
		BaseTask: domain.BaseTask{
			ProjectID: "test-proj",
			TaskID:    "task-123",
		},
	}

	ctx := context.Background()

	// Expectations
	mockRepo.On("DownloadProject", ctx, "test-proj", mock.AnythingOfType("string")).
		Return(errors.New("minio error"))

	// Execute
	output, err := uc.Execute(ctx, logger, task)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to download")
	assert.Nil(t, output)
}

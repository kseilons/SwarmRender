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
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	allowedDevices := []string{"CUDA", "OPTIX"}
	uc := usecase.NewRenderUseCase(mockRenderer, mockRepo, allowedDevices, 4)

	task := domain.RenderTask{
		CommonTask: domain.CommonTask{
			StartFrame: 1,
			EndFrame:   1,
			Options: domain.RenderOptions{
				Engine:       domain.EngineCycles,
				CyclesDevice: domain.DeviceCUDA,
			},
		},
		ProjectID: "test-proj",
		BlendFile: "scene.blend",
	}

	ctx := context.Background()

	// Expectations
	mockRepo.On("DownloadProject", ctx, "test-proj", mock.AnythingOfType("string")).Return(nil)
	mockRenderer.On("Render", ctx, logger, mock.MatchedBy(func(job domain.RenderJob) bool {
		return job.Options.Threads == 4
	})).Return([]byte("rendered-data"), nil)
	mockRepo.On("UploadResult", ctx, "test-proj", mock.AnythingOfType("string")).Return(nil)

	// Execute
	output, err := uc.Execute(ctx, logger, task)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, []byte("rendered-data"), output)
}

func TestRenderUseCase_Execute_ValidationFailure(t *testing.T) {
	// Setup
	mockRenderer := mocks.NewRenderer(t)
	mockRepo := mocks.NewProjectRepository(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	allowedDevices := []string{"CPU"}
	uc := usecase.NewRenderUseCase(mockRenderer, mockRepo, allowedDevices, 0)

	task := domain.RenderTask{
		CommonTask: domain.CommonTask{
			Options: domain.RenderOptions{
				Engine:       domain.EngineCycles,
				CyclesDevice: domain.DeviceCUDA, // Not allowed
			},
		},
		ProjectID: "test-proj",
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
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	uc := usecase.NewRenderUseCase(mockRenderer, mockRepo, nil, 0)

	task := domain.RenderTask{
		ProjectID: "test-proj",
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

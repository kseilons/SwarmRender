package usecase_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
		MainFileURL: "scene.blend",
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
	})).Return([]byte("rendered-data"), nil)
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
	mockRenderer.On("Render", ctx, logger, mock.Anything).Return(nil, errors.New("blender crash"))
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
	mockRenderer.On("Render", ctx, logger, mock.Anything).Return([]byte("ok"), nil)
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

func TestRenderUseCase_Execute_UploadFailure(t *testing.T) {
	mockRenderer := mocks.NewRenderer(t)
	mockRepo := mocks.NewProjectRepository(t)
	mockProducer := mocks.NewResultProducer(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	uc := usecase.NewRenderUseCase(mockRenderer, mockRepo, mockProducer, nil, 0)
	task := domain.RenderTask{BaseTask: domain.BaseTask{ProjectID: "p1", TaskID: "t1"}}
	ctx := context.Background()

	mockRepo.On("DownloadProject", ctx, "p1", mock.Anything).Return(nil)
	mockRenderer.On("Render", ctx, logger, mock.Anything).Return([]byte("ok"), nil)
	mockRepo.On("UploadResult", ctx, "p1", mock.Anything).Return(errors.New("s3 upload fail"))
	mockProducer.On("ReportResult", ctx, mock.MatchedBy(func(res domain.RenderResult) bool {
		return res.Status == domain.StatusFailed && res.Error != ""
	})).Return(nil)

	_, err := uc.Execute(ctx, logger, task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to upload")
}

func TestRenderUseCase_InternalHelpers(t *testing.T) {
	uc := usecase.NewRenderUseCase(nil, nil, nil, []string{"CUDA"}, 0)

	t.Run("truncateLogs", func(t *testing.T) {
		s := "short"
		assert.Equal(t, s, uc.ExportTruncateLogs(s, 10))

		long := "this is a long string"
		truncated := uc.ExportTruncateLogs(long, 5)
		assert.Contains(t, truncated, "[TRUNCATED]")
		assert.Equal(t, "...[TRUNCATED]...\ntring", truncated)
	})

	t.Run("isDeviceAllowed", func(t *testing.T) {
		assert.True(t, uc.ExportIsDeviceAllowed("CUDA"))
		assert.False(t, uc.ExportIsDeviceAllowed("CPU"))

		ucAll := usecase.NewRenderUseCase(nil, nil, nil, nil, 0)
		assert.True(t, ucAll.ExportIsDeviceAllowed("anything"))
	})
}

func TestRenderUseCase_Execute_WithProgress(t *testing.T) {
	mockRenderer := mocks.NewRenderer(t)
	mockRepo := mocks.NewProjectRepository(t)
	mockProducer := mocks.NewResultProducer(t)
	logger := slog.Default()

	uc := usecase.NewRenderUseCase(mockRenderer, mockRepo, mockProducer, nil, 0)
	task := domain.RenderTask{
		BaseTask:   domain.BaseTask{ProjectID: "p-progress", TaskID: "t-progress"},
		CommonTask: domain.CommonTask{StartFrame: 1, EndFrame: 5},
	}
	ctx := context.Background()

	mockRepo.On("DownloadProject", ctx, "p-progress", mock.Anything).Return(nil)

	// We want to simulate renderer writing files to the output directory
	mockRenderer.On("Render", ctx, logger, mock.Anything).Run(func(args mock.Arguments) {
		job := args.Get(2).(domain.RenderJob)
		// Extract directory from OutputPath (which might have a prefix)
		dir := filepath.Dir(job.OutputPath)
		if strings.HasSuffix(job.OutputPath, string(os.PathSeparator)) {
			dir = job.OutputPath
		}

		// Create a file to trigger fsnotify
		framePath := filepath.Join(dir, "frame_0001.png")
		_ = os.WriteFile(framePath, []byte("data"), 0644)
		// Give some time for fsnotify to catch up
		time.Sleep(100 * time.Millisecond)
	}).Return([]byte("ok"), nil)

	mockRepo.On("UploadResult", ctx, "p-progress", mock.Anything).Return(nil)

	// Expect at least one progress report
	mockProducer.On("ReportResult", ctx, mock.MatchedBy(func(res domain.RenderResult) bool {
		return res.Status == domain.StatusProcessing && res.CurrentFrame == 1
	})).Return(nil).Once()

	// And the final completion report
	mockProducer.On("ReportResult", ctx, mock.MatchedBy(func(res domain.RenderResult) bool {
		return res.Status == domain.StatusCompleted
	})).Return(nil).Once()

	_, err := uc.Execute(ctx, logger, task)
	assert.NoError(t, err)
}

func TestRenderUseCase_PrepareJob_WithScene(t *testing.T) {
	uc := usecase.NewRenderUseCase(nil, nil, nil, nil, 8)
	task := domain.RenderTask{
		CommonTask: domain.CommonTask{
			Options: domain.RenderOptions{Scene: "MyScene"},
		},
	}
	job := uc.ExportPrepareJob(task, "blend", "out")
	assert.Contains(t, job.OutputPath, "MyScene_")
	assert.Equal(t, 8, job.Options.Threads)
}

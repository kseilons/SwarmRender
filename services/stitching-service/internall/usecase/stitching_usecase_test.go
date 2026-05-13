package usecase_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"kseilons/stitching-service/internall/domain"
	"kseilons/stitching-service/internall/usecase"
	"kseilons/stitching-service/internall/usecase/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupUseCase(t *testing.T) (*usecase.StitchingUseCase, *mocks.ProjectRepository, *mocks.ResultProducer, *mocks.Stitcher, *slog.Logger) {
	mockRepo := mocks.NewProjectRepository(t)
	mockProducer := mocks.NewResultProducer(t)
	mockStitcher := mocks.NewStitcher(t)
	logger := slog.Default()
	uc := usecase.NewStitchingUseCase(mockRepo, mockProducer, mockStitcher)
	return uc, mockRepo, mockProducer, mockStitcher, logger
}

func TestStitchingUseCase_Execute_Success(t *testing.T) {
	uc, mockRepo, mockProducer, mockStitcher, logger := setupUseCase(t)
	ctx := context.Background()
	task := domain.StitchingTask{BaseTask: domain.BaseTask{ProjectID: "job-123", TaskID: "stitch-123"}}

	mockRepo.On("DownloadFrames", ctx, "job-123", mock.Anything).Return(nil).Once().Run(func(args mock.Arguments) {
		dir := args.String(2)
		os.WriteFile(filepath.Join(dir, "frame_0001.png"), []byte("data"), 0644)
	})

	mockStitcher.On("Stitch", ctx, logger, mock.Anything, mock.Anything, []string{"frame_0001.png"}).Return(nil).Once()
	mockRepo.On("UploadFinalVideo", ctx, "job-123", mock.Anything).Return("job-123_final.mp4", nil).Once()

	mockProducer.On("ReportResult", ctx, mock.MatchedBy(func(res domain.RenderResult) bool {
		return res.Status == domain.StatusCompleted && res.ResultURL == "job-123_final.mp4"
	})).Return(nil).Once()

	err := uc.Execute(ctx, logger, task)
	assert.NoError(t, err)
}

func TestStitchingUseCase_Execute_NoFrames(t *testing.T) {
	uc, mockRepo, mockProducer, _, logger := setupUseCase(t)
	ctx := context.Background()
	task := domain.StitchingTask{BaseTask: domain.BaseTask{ProjectID: "job-123", TaskID: "stitch-123"}}

	mockRepo.On("DownloadFrames", ctx, "job-123", mock.Anything).Return(nil).Once()
	mockProducer.On("ReportResult", ctx, mock.Anything).Return(nil).Once()

	err := uc.Execute(ctx, logger, task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no frames found")
}

func TestStitchingUseCase_Execute_DownloadError(t *testing.T) {
	uc, mockRepo, mockProducer, _, logger := setupUseCase(t)
	ctx := context.Background()
	task := domain.StitchingTask{BaseTask: domain.BaseTask{ProjectID: "job-123", TaskID: "stitch-123"}}

	mockRepo.On("DownloadFrames", ctx, "job-123", mock.Anything).Return(errors.New("download failed")).Once()
	mockProducer.On("ReportResult", ctx, mock.Anything).Return(nil).Once()

	err := uc.Execute(ctx, logger, task)
	assert.Error(t, err)
}

func TestStitchingUseCase_Execute_StitchError(t *testing.T) {
	uc, mockRepo, mockProducer, mockStitcher, logger := setupUseCase(t)
	ctx := context.Background()
	task := domain.StitchingTask{BaseTask: domain.BaseTask{ProjectID: "job-123", TaskID: "stitch-123"}}

	mockRepo.On("DownloadFrames", ctx, "job-123", mock.Anything).Return(nil).Once().Run(func(args mock.Arguments) {
		dir := args.String(2)
		os.WriteFile(filepath.Join(dir, "frame_0001.png"), []byte("data"), 0644)
	})

	mockStitcher.On("Stitch", ctx, logger, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("ffmpeg crash")).Once()
	mockProducer.On("ReportResult", ctx, mock.Anything).Return(nil).Once()

	err := uc.Execute(ctx, logger, task)
	assert.Error(t, err)
}

func TestStitchingUseCase_Execute_UploadError(t *testing.T) {
	uc, mockRepo, mockProducer, mockStitcher, logger := setupUseCase(t)
	ctx := context.Background()
	task := domain.StitchingTask{BaseTask: domain.BaseTask{ProjectID: "job-123", TaskID: "stitch-123"}}

	mockRepo.On("DownloadFrames", ctx, "job-123", mock.Anything).Return(nil).Once().Run(func(args mock.Arguments) {
		dir := args.String(2)
		os.WriteFile(filepath.Join(dir, "frame_0001.png"), []byte("data"), 0644)
	})

	mockStitcher.On("Stitch", ctx, logger, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	mockRepo.On("UploadFinalVideo", ctx, "job-123", mock.Anything).Return("", errors.New("upload failed")).Once()
	mockProducer.On("ReportResult", ctx, mock.Anything).Return(nil).Once()

	err := uc.Execute(ctx, logger, task)
	assert.Error(t, err)
}

func TestStitchingUseCase_Execute_ReportResultError(t *testing.T) {
	uc, mockRepo, mockProducer, mockStitcher, logger := setupUseCase(t)
	ctx := context.Background()
	task := domain.StitchingTask{BaseTask: domain.BaseTask{ProjectID: "job-123", TaskID: "stitch-123"}}

	mockRepo.On("DownloadFrames", ctx, "job-123", mock.Anything).Return(nil).Once().Run(func(args mock.Arguments) {
		dir := args.String(2)
		os.WriteFile(filepath.Join(dir, "frame_0001.png"), []byte("data"), 0644)
	})

	mockStitcher.On("Stitch", ctx, logger, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	mockRepo.On("UploadFinalVideo", ctx, "job-123", mock.Anything).Return("job-123_final.mp4", nil).Once()
	mockProducer.On("ReportResult", ctx, mock.Anything).Return(errors.New("kafka down")).Once()

	err := uc.Execute(ctx, logger, task)
	assert.Error(t, err)
}

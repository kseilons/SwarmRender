package usecase_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"kseilons/stitching-service/internall/domain"
	"kseilons/stitching-service/internall/usecase"
	"kseilons/stitching-service/internall/usecase/mocks"
)

func TestStitchingUseCase_Execute(t *testing.T) {
	mockRepo := mocks.NewProjectRepository(t)
	mockProducer := mocks.NewResultProducer(t)
	mockStitcher := mocks.NewStitcher(t)
	logger := slog.Default()

	uc := usecase.NewStitchingUseCase(mockRepo, mockProducer, mockStitcher)

	task := domain.StitchingTask{
		BaseTask: domain.BaseTask{
			ProjectID: "job-123",
			TaskID:    "stitch-123",
		},
	}

	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		// Mock frame download
		mockRepo.On("DownloadFrames", ctx, "job-123", mock.Anything).Return(nil).Once().Run(func(args mock.Arguments) {
			dir := args.String(2)
			os.WriteFile(filepath.Join(dir, "frame_0001.png"), []byte("data"), 0644)
		})

		mockStitcher.On("Stitch", ctx, logger, mock.Anything, mock.Anything, []string{"frame_0001.png"}).Return(nil).Once()
		mockRepo.On("UploadFinalVideo", ctx, "job-123", mock.Anything).Return("s3://results/job-123/final.mp4", nil).Once()
		
		mockProducer.On("ReportResult", ctx, mock.MatchedBy(func(res domain.RenderResult) bool {
			return res.Status == domain.StatusCompleted && res.ResultPath == "s3://results/job-123/final.mp4"
		})).Return(nil).Once()

		err := uc.Execute(ctx, logger, task)
		assert.NoError(t, err)
	})

	t.Run("NoFrames", func(t *testing.T) {
		mockRepo.On("DownloadFrames", ctx, "job-123", mock.Anything).Return(nil).Once()
		
		mockProducer.On("ReportResult", ctx, mock.MatchedBy(func(res domain.RenderResult) bool {
			return res.Status == domain.StatusFailed && res.Error != ""
		})).Return(nil).Once()

		err := uc.Execute(ctx, logger, task)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no frames found")
	})

	t.Run("DownloadError", func(t *testing.T) {
		mockRepo.On("DownloadFrames", ctx, "job-123", mock.Anything).Return(errors.New("s3 error")).Once()
		
		mockProducer.On("ReportResult", ctx, mock.MatchedBy(func(res domain.RenderResult) bool {
			return res.Status == domain.StatusFailed && res.Error == "s3 error"
		})).Return(nil).Once()

		err := uc.Execute(ctx, logger, task)
		assert.Error(t, err)
	})

	t.Run("StitchError", func(t *testing.T) {
		mockRepo.On("DownloadFrames", ctx, "job-123", mock.Anything).Return(nil).Once().Run(func(args mock.Arguments) {
			dir := args.String(2)
			os.WriteFile(filepath.Join(dir, "frame_0001.png"), []byte("data"), 0644)
		})

		mockStitcher.On("Stitch", ctx, logger, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("ffmpeg crash")).Once()
		
		mockProducer.On("ReportResult", ctx, mock.MatchedBy(func(res domain.RenderResult) bool {
			return res.Status == domain.StatusFailed && res.Error == "ffmpeg crash"
		})).Return(nil).Once()

		err := uc.Execute(ctx, logger, task)
		assert.Error(t, err)
	})
}

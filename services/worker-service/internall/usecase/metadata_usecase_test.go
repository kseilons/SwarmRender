package usecase_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"kseilons/worker-service/internall/domain"
	"kseilons/worker-service/internall/usecase"
	"kseilons/worker-service/internall/usecase/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMetadataUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	task := domain.MetadataTask{
		BaseTask: domain.BaseTask{
			ProjectID: "p1",
			TaskID:    "t1",
		},
		MainFileURL: "p1_sources/main.blend",
	}

	t.Run("Success", func(t *testing.T) {
		mockRenderer := mocks.NewMetadataRenderer(t)
		mockRepo := mocks.NewProjectRepository(t)
		mockProducer := mocks.NewResultProducer(t)
		uc := usecase.NewMetadataUseCase(mockRenderer, mockRepo, mockProducer)

		mockRepo.On("DownloadProject", ctx, "p1", mock.Anything).Return(nil).Once()
		mockRenderer.On("GetMetadata", ctx, mock.Anything).Return(domain.ProjectMetadata{
			FrameStart: 1,
			FrameEnd:   100,
		}, nil).Once()
		mockProducer.On("ReportResult", ctx, mock.MatchedBy(func(res domain.RenderResult) bool {
			return res.Status == domain.StatusCompleted && res.Metadata.FrameEnd == 100
		})).Return(nil).Once()

		err := uc.Execute(ctx, logger, task)
		assert.NoError(t, err)
	})

	t.Run("DownloadFailure", func(t *testing.T) {
		mockRenderer := mocks.NewMetadataRenderer(t)
		mockRepo := mocks.NewProjectRepository(t)
		mockProducer := mocks.NewResultProducer(t)
		uc := usecase.NewMetadataUseCase(mockRenderer, mockRepo, mockProducer)

		mockRepo.On("DownloadProject", ctx, "p1", mock.Anything).Return(errors.New("down fail")).Once()
		mockProducer.On("ReportResult", ctx, mock.MatchedBy(func(res domain.RenderResult) bool {
			return res.Status == domain.StatusFailed && res.Error == "down fail"
		})).Return(nil).Once()

		err := uc.Execute(ctx, logger, task)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "download")
	})

	t.Run("ExtractionFailure", func(t *testing.T) {
		mockRenderer := mocks.NewMetadataRenderer(t)
		mockRepo := mocks.NewProjectRepository(t)
		mockProducer := mocks.NewResultProducer(t)
		uc := usecase.NewMetadataUseCase(mockRenderer, mockRepo, mockProducer)

		mockRepo.On("DownloadProject", ctx, "p1", mock.Anything).Return(nil).Once()
		mockRenderer.On("GetMetadata", ctx, mock.Anything).Return(domain.ProjectMetadata{}, errors.New("extract fail")).Once()
		mockProducer.On("ReportResult", ctx, mock.MatchedBy(func(res domain.RenderResult) bool {
			return res.Status == domain.StatusFailed && res.Error == "extract fail"
		})).Return(nil).Once()

		err := uc.Execute(ctx, logger, task)
		assert.Error(t, err)
		assert.Equal(t, "extract fail", err.Error())
	})
}

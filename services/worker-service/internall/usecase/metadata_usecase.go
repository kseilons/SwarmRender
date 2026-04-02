package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"kseilons/worker-service/internall/domain"
)

type MetadataRenderer interface {
	GetMetadata(ctx context.Context, blendPath string) (domain.ProjectMetadata, error)
}

type MetadataUseCase struct {
	renderer       MetadataRenderer
	projectRepo    ProjectRepository
	resultProducer ResultProducer
}

func NewMetadataUseCase(renderer MetadataRenderer, projectRepo ProjectRepository, resultProducer ResultProducer) *MetadataUseCase {
	return &MetadataUseCase{
		renderer:       renderer,
		projectRepo:    projectRepo,
		resultProducer: resultProducer,
	}
}

func (uc *MetadataUseCase) Execute(ctx context.Context, logger *slog.Logger, task domain.MetadataTask) error {
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("metadata-%s-*", task.ProjectID))
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	logger.Info("downloading project for metadata extraction", slog.String("project_id", task.ProjectID))
	if err := uc.projectRepo.DownloadProject(ctx, task.ProjectID, tempDir); err != nil {
		uc.reportError(ctx, logger, task, err)
		return fmt.Errorf("failed to download project: %w", err)
	}

	blendPath := filepath.Join(tempDir, extractRelativePath(task.ProjectID, task.MainFileURL))
	logger.Info("extracting metadata", slog.String("blend_path", blendPath))

	metadata, err := uc.renderer.GetMetadata(ctx, blendPath)
	if err != nil {
		uc.reportError(ctx, logger, task, err)
		return err
	}

	logger.Info("metadata extracted successfully",
		slog.Int("start_frame", metadata.FrameStart),
		slog.Int("end_frame", metadata.FrameEnd),
	)

	err = uc.resultProducer.ReportResult(ctx, domain.RenderResult{
		TaskID:    task.TaskID,
		ProjectID: task.ProjectID,
		Type:      domain.TypeMetadata,
		Status:    domain.StatusCompleted,
		WorkerID:  os.Getenv("HOSTNAME"),
		Metadata:  &metadata,
	})
	if err != nil {
		logger.Error("failed to report metadata result", slog.Any("error", err))
		return err
	}

	return nil
}

func (uc *MetadataUseCase) reportError(ctx context.Context, logger *slog.Logger, task domain.MetadataTask, err error) {
	reportErr := uc.resultProducer.ReportResult(ctx, domain.RenderResult{
		TaskID:    task.TaskID,
		ProjectID: task.ProjectID,
		Type:      domain.TypeMetadata,
		Status:    domain.StatusFailed,
		WorkerID:  os.Getenv("HOSTNAME"),
		Error:     err.Error(),
	})
	if reportErr != nil {
		logger.Error("failed to report failure result", slog.Any("error", reportErr))
	}
}

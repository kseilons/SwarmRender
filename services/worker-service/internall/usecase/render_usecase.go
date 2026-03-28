package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"kseilons/worker-service/internall/domain"
)

type ProjectRepository interface {
	DownloadProject(ctx context.Context, projectID string, destDir string) error
	UploadResult(ctx context.Context, projectID string, resultDir string) error
}

type Renderer interface {
	Render(ctx context.Context, logger *slog.Logger, job domain.RenderJob) ([]byte, error)
}

type RenderUseCase struct {
	renderer       Renderer
	projectRepo    ProjectRepository
	allowedDevices []string
	defaultThreads int
}

func NewRenderUseCase(renderer Renderer, projectRepo ProjectRepository, allowedDevices []string, defaultThreads int) *RenderUseCase {
	return &RenderUseCase{
		renderer:       renderer,
		projectRepo:    projectRepo,
		allowedDevices: allowedDevices,
		defaultThreads: defaultThreads,
	}
}

// Execute performs validation and orchestrates the render task.
func (uc *RenderUseCase) Execute(ctx context.Context, logger *slog.Logger, task domain.RenderTask) ([]byte, error) {
	if err := uc.validate(task); err != nil {
		logger.Error("policy validation failed", slog.Any("error", err))
		return nil, err
	}

	tempDir, err := os.MkdirTemp("", fmt.Sprintf("render-%s-*", task.ProjectID))
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := uc.downloadProject(ctx, logger, task.ProjectID, tempDir); err != nil {
		return nil, err
	}

	blendPath := filepath.Join(tempDir, task.BlendFile)
	outputDir := filepath.Join(tempDir, "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output dir: %w", err)
	}

	job := uc.prepareJob(task, blendPath, outputDir)

	logger.Info("calling renderer",
		slog.String("blend_path", job.BlendPath),
		slog.String("output_path", job.OutputPath),
	)

	renderOutput, err := uc.renderer.Render(ctx, logger, job)
	if err != nil {
		return renderOutput, err
	}

	if err := uc.uploadResults(ctx, logger, task.ProjectID, outputDir); err != nil {
		return renderOutput, err
	}

	return renderOutput, nil
}

func (uc *RenderUseCase) validate(task domain.RenderTask) error {
	if task.Options.Engine == domain.EngineCycles {
		if !uc.isDeviceAllowed(string(task.Options.CyclesDevice)) {
			return fmt.Errorf("device %s is not allowed on this worker", task.Options.CyclesDevice)
		}
	}
	return nil
}

func (uc *RenderUseCase) downloadProject(ctx context.Context, logger *slog.Logger, projectID, tempDir string) error {
	logger.Info("downloading project sources", slog.String("project_id", projectID))
	if err := uc.projectRepo.DownloadProject(ctx, projectID, tempDir); err != nil {
		return fmt.Errorf("failed to download project: %w", err)
	}
	return nil
}

func (uc *RenderUseCase) prepareJob(task domain.RenderTask, blendPath, outputDir string) domain.RenderJob {
	outputPrefix := ""
	if task.Options.Scene != "" {
		outputPrefix = fmt.Sprintf("%s_", task.Options.Scene)
	}
	outputPath := filepath.Join(outputDir, outputPrefix)
	if outputPrefix == "" {
		outputPath += string(os.PathSeparator)
	}

	job := domain.RenderJob{
		CommonTask: task.CommonTask,
		BlendPath:  blendPath,
		OutputPath: outputPath,
	}

	if job.Options.Threads == 0 {
		job.Options.Threads = uc.defaultThreads
	}
	return job
}

func (uc *RenderUseCase) uploadResults(ctx context.Context, logger *slog.Logger, projectID, outputDir string) error {
	logger.Info("uploading render results", slog.String("project_id", projectID))

	files, err := os.ReadDir(outputDir)
	if err != nil {
		logger.Warn("failed to read output directory", slog.String("dir", outputDir), slog.Any("error", err))
	} else {
		logger.Info("found files in output directory", slog.Int("count", len(files)), slog.String("dir", outputDir))
	}

	if err := uc.projectRepo.UploadResult(ctx, projectID, outputDir); err != nil {
		return fmt.Errorf("failed to upload results: %w", err)
	}
	return nil
}

func (uc *RenderUseCase) isDeviceAllowed(device string) bool {
	if len(uc.allowedDevices) == 0 {
		return true
	}
	for _, a := range uc.allowedDevices {
		if a == device {
			return true
		}
	}
	return false
}

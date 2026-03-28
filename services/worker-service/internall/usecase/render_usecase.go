package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"kseilons/worker-service/internall/domain"
)

type Renderer interface {
	Render(ctx context.Context, logger *slog.Logger, job domain.RenderJob) ([]byte, error)
}

type RenderUseCase struct {
	renderer       Renderer
	allowedDevices []string
	defaultThreads int
}

func NewRenderUseCase(renderer Renderer, allowedDevices []string, defaultThreads int) *RenderUseCase {
	return &RenderUseCase{
		renderer:       renderer,
		allowedDevices: allowedDevices,
		defaultThreads: defaultThreads,
	}
}

// Execute performs validation and orchestrates the render task.
func (uc *RenderUseCase) Execute(ctx context.Context, logger *slog.Logger, task domain.RenderTask) ([]byte, error) {
	// Business Validation (Hardware Policy)
	if task.Options.Engine == domain.EngineCycles {
		if !uc.isDeviceAllowed(string(task.Options.CyclesDevice)) {
			err := fmt.Errorf("device %s is not allowed on this worker", task.Options.CyclesDevice)
			logger.Error("policy validation failed", slog.Any("error", err))
			return nil, err
		}
	}

	// TODO: Download project from MinIO based on task.ProjectID
	// For now, we'll assume a local path mapping or mock it.
	blendPath := fmt.Sprintf("/tmp/renders/%s/%s", task.ProjectID, task.BlendFile)
	outputPath := fmt.Sprintf("/tmp/renders/%s/output/", task.ProjectID)

	job := domain.RenderJob{
		CommonTask: task.CommonTask,
		BlendPath:  blendPath,
		OutputPath: outputPath,
	}

	// Apply hardware settings from config
	if job.Options.Threads == 0 {
		job.Options.Threads = uc.defaultThreads
	}

	// Call Infrastructure Layer
	logger.Info("policy validation passed, calling renderer",
		slog.String("blend_path", job.BlendPath),
	)
	return uc.renderer.Render(ctx, logger, job)
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

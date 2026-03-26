package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"kseilons/worker-service/internall/domain"
)

type Renderer interface {
	Render(ctx context.Context, logger *slog.Logger, task domain.RenderTask) ([]byte, error)
}

type RenderUseCase struct {
	renderer       Renderer
	allowedDevices []string
}

func NewRenderUseCase(renderer Renderer, allowedDevices []string) *RenderUseCase {
	return &RenderUseCase{
		renderer:       renderer,
		allowedDevices: allowedDevices,
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

	// Call Infrastructure Layer
	logger.Info("policy validation passed, calling renderer")
	return uc.renderer.Render(ctx, logger, task)
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

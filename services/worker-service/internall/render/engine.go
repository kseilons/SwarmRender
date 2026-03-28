package render

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"

	"kseilons/worker-service/internall/domain"
)

type Engine struct {
	executablePath string
}

func NewEngine(executablePath string) *Engine {
	return &Engine{executablePath: executablePath}
}

func (e *Engine) Render(ctx context.Context, logger *slog.Logger, job domain.RenderJob) ([]byte, error) {
	if job.BlendPath == "" {
		err := fmt.Errorf("blend file path is required")
		logger.Error("validation failed", slog.Any("error", err))
		return nil, err
	}

	args := []string{}

	if job.Options.Background {
		args = append(args, "-b")
	}

	args = append(args, job.BlendPath)

	if job.Options.Engine != "" {
		args = append(args, "-E", string(job.Options.Engine))
	}

	if job.Options.Threads >= 0 {
		args = append(args, "-t", strconv.Itoa(job.Options.Threads))
	}

	if job.OutputPath != "" {
		args = append(args, "-o", job.OutputPath)
	}

	if job.Options.Format != "" {
		args = append(args, "-F", string(job.Options.Format))
	}

	if job.Options.Scene != "" {
		args = append(args, "-S", job.Options.Scene)
	}

	if job.StartFrame == job.EndFrame {
		args = append(args, "-f", strconv.Itoa(job.StartFrame))
	} else {
		args = append(args, "-s", strconv.Itoa(job.StartFrame), "-e", strconv.Itoa(job.EndFrame), "-a")
	}

	if job.Options.Engine == domain.EngineCycles {
		args = append(args, "--")
		if job.Options.CyclesDevice != "" {
			device := string(job.Options.CyclesDevice)
			if job.Options.UseCPU && device != string(domain.DeviceCPU) {
				// Append +CPU to a GPU device to render on both CPU and GPU.
				device += "+CPU"
			}
			args = append(args, "--cycles-device", device)
		}
	}

	logger.Debug("executing blender command",
		slog.String("path", e.executablePath),
		slog.Any("args", args),
	)

	cmd := exec.CommandContext(ctx, e.executablePath, args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("render failed",
			slog.Any("error", err),
			slog.String("output", string(output)),
		)
		return output, err
	}

	logger.Info("render completed successfully",
		slog.Int("output_size", len(output)),
	)

	return output, nil
}

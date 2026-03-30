package render

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"

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

	args := e.BuildArgs(job)

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

func (e *Engine) GetMetadata(ctx context.Context, blendPath string) (domain.ProjectMetadata, error) {
	pyExpr := "import bpy; import json; print('FRAME_INFO:' + json.dumps({'frame_start': bpy.context.scene.frame_start, 'frame_end': bpy.context.scene.frame_end}))"
	args := []string{"-b", blendPath, "--python-expr", pyExpr}

	cmd := exec.CommandContext(ctx, e.executablePath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return domain.ProjectMetadata{}, fmt.Errorf("blender metadata extraction failed: %w (output: %s)", err, string(output))
	}

	lines := strings.SplitSeq(string(output), "\n")
	for line := range lines {
		if after, ok := strings.CutPrefix(line, "FRAME_INFO:"); ok {
			jsonStr := after
			var metadata domain.ProjectMetadata
			if err := json.Unmarshal([]byte(jsonStr), &metadata); err != nil {
				return domain.ProjectMetadata{}, fmt.Errorf("failed to parse metadata JSON: %w", err)
			}
			return metadata, nil
		}
	}

	return domain.ProjectMetadata{}, fmt.Errorf("could not find FRAME_INFO in blender output")
}

func (e *Engine) BuildArgs(job domain.RenderJob) []string {
	args := []string{}

	if job.Options.Background {
		args = append(args, "-b")
	}

	if job.BlendPath != "" {
		args = append(args, job.BlendPath)
	}

	if job.Options.Engine != "" {
		args = append(args, "-E", string(job.Options.Engine))
	}

	if job.Options.Threads > 0 {
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
		if job.StartFrame > 0 {
			args = append(args, "-f", strconv.Itoa(job.StartFrame))
		}
	} else {
		args = append(args, "-s", strconv.Itoa(job.StartFrame), "-e", strconv.Itoa(job.EndFrame), "-a")
	}

	if job.Options.Engine == domain.EngineCycles {
		args = append(args, "--")
		if job.Options.CyclesDevice != "" {
			device := string(job.Options.CyclesDevice)
			if job.Options.UseCPU && device != string(domain.DeviceCPU) {
				device += "+CPU"
			}
			args = append(args, "--cycles-device", device)
		}
	}

	return args
}

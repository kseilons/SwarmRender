package render

import (
	"context"
	"fmt"
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

func (e *Engine) Render(ctx context.Context, task domain.RenderTask) ([]byte, error) {
	args := []string{}

	if task.Options.Background {
		args = append(args, "-b")
	}

	if task.BlendPath == "" {
		return nil, fmt.Errorf("blend file path is required")
	}
	args = append(args, task.BlendPath)

	if task.Options.Engine != "" {
		args = append(args, "-E", string(task.Options.Engine))
	}

	if task.Options.Threads >= 0 {
		args = append(args, "-t", strconv.Itoa(task.Options.Threads))
	}

	if task.OutputPath != "" {
		args = append(args, "-o", task.OutputPath)
	}

	if task.Options.Format != "" {
		args = append(args, "-F", string(task.Options.Format))
	}

	if task.Options.Scene != "" {
		args = append(args, "-S", task.Options.Scene)
	}

	if task.StartFrame == task.EndFrame {
		args = append(args, "-f", strconv.Itoa(task.StartFrame))
	} else {
		args = append(args, "-s", strconv.Itoa(task.StartFrame), "-e", strconv.Itoa(task.EndFrame), "-a")
	}

	if task.Options.Engine == domain.EngineCycles {
		args = append(args, "--")
		if task.Options.CyclesDevice != "" {
			device := string(task.Options.CyclesDevice)
			if task.Options.UseCPU && device != string(domain.DeviceCPU) {
				// Append +CPU to a GPU device to render on both CPU and GPU.
				device += "+CPU"
			}
			args = append(args, "--cycles-device", device)
		}
	}

	cmd := exec.CommandContext(ctx, e.executablePath, args...)

	return cmd.CombinedOutput()
}

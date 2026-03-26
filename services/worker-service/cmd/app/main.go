package main

import (
	"context"
	"fmt"
	"kseilons/worker-service/internall/domain"
	"kseilons/worker-service/internall/render"
	"kseilons/worker-service/pkg/logger"
)

func main() {
	log := logger.SetupLogger(logger.EnvLocal)

	ctx := context.Background()

	// Inject logger into engine
	renderer := render.NewEngine("blender")

	task := domain.RenderTask{
		BlendPath: "C:\\doc\\Диплом\\blender-project\\basketball.blend",
		Options: domain.RenderOptions{
			Engine:       domain.EngineCycles,
			UseCPU:       true,
			CyclesDevice: domain.DeviceCPU,
			Threads:      0,
			Format:       domain.FormatPNG,
			Scene:        "",
			Background:   true,
		},
		StartFrame: 1,
		EndFrame:   10,
		OutputPath: "C:\\doc\\Диплом\\blender-project\\output\\",
	}

	// Use LogAttrs to create a contextual logger effortlessly
	taskLog := log.With(task.LogAttrs()...)

	taskLog.Info("starting application")

	output, err := renderer.Render(ctx, taskLog, task)
	if err != nil {
		taskLog.Error("application error", logger.Err(err))
		return
	}

	taskLog.Info("task completed")
	fmt.Println(string(output))
}

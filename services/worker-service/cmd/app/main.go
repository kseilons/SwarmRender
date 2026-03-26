package main

import (
	"context"
	"fmt"
	"kseilons/worker-service/internall/config"
	"kseilons/worker-service/internall/domain"
	"kseilons/worker-service/internall/render"
	"kseilons/worker-service/internall/usecase"
	"kseilons/worker-service/pkg/logger"
)

func main() {
	cfg := config.MustLoad()

	log := logger.SetupLogger(cfg.Env)
	log.Info("starting application", "env", cfg.Env)

	ctx := context.Background()

	engine := render.NewEngine(cfg.Blender.Path)

	renderUseCase := usecase.NewRenderUseCase(engine, cfg.Blender.AllowedDevices)

	task := domain.RenderTask{
		BlendPath: "C:\\doc\\Диплом\\blender-project\\basketball.blend",
		Options: domain.RenderOptions{
			Engine:       domain.EngineCycles,
			UseCPU:       true,
			CyclesDevice: domain.DeviceCUDA,
			Threads:      cfg.Blender.Threads, // Use threads from nested config
			Format:       domain.FormatPNG,
			Scene:        "",
			Background:   true,
		},
		StartFrame: 1,
		EndFrame:   10,
		OutputPath: "C:\\doc\\Диплом\\blender-project\\output\\",
	}

	taskLog := log.With(task.LogAttrs()...)

	taskLog.Info("starting task execution via usecase")

	output, err := renderUseCase.Execute(ctx, taskLog, task)
	if err != nil {
		taskLog.Error("application error", logger.Err(err))
		return
	}

	taskLog.Info("task completed")
	fmt.Println(string(output))
}

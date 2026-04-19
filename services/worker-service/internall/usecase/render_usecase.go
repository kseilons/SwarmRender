package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"kseilons/worker-service/internall/domain"
	"kseilons/worker-service/pkg/metrics"

	"github.com/fsnotify/fsnotify"
)

type ProjectRepository interface {
	DownloadProject(ctx context.Context, projectID string, destDir string) error
	UploadResult(ctx context.Context, projectID string, resultDir string) error
}

type Renderer interface {
	Render(ctx context.Context, logger *slog.Logger, job domain.RenderJob) ([]byte, error)
}

type ResultProducer interface {
	ReportResult(ctx context.Context, result domain.RenderResult) error
}

type RenderUseCase struct {
	renderer       Renderer
	projectRepo    ProjectRepository
	resultProducer ResultProducer
	allowedDevices []string
	defaultThreads int
}

func NewRenderUseCase(renderer Renderer, projectRepo ProjectRepository, resultProducer ResultProducer, allowedDevices []string, defaultThreads int) *RenderUseCase {
	return &RenderUseCase{
		renderer:       renderer,
		projectRepo:    projectRepo,
		resultProducer: resultProducer,
		allowedDevices: allowedDevices,
		defaultThreads: defaultThreads,
	}
}

func (uc *RenderUseCase) getWorkerID() string {
	hostname, _ := os.Hostname()
	if hostname == "" {
		return "unknown-worker"
	}
	return hostname
}

// Execute performs validation and orchestrates the render task.
func (uc *RenderUseCase) Execute(ctx context.Context, logger *slog.Logger, task domain.RenderTask) ([]byte, error) {
	if err := uc.validate(task); err != nil {
		logger.Error("policy validation failed", slog.Any("error", err))
		return nil, err
	}

	// Calculate queue wait time
	if !task.CreatedAt.IsZero() {
		metrics.QueueWaitTime.Observe(time.Since(task.CreatedAt).Seconds())
	}

	metrics.ActiveJobs.Inc()
	defer metrics.ActiveJobs.Dec()

	tempDir, err := os.MkdirTemp("", fmt.Sprintf("render-%s-*", task.ProjectID))
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	downloadStart := time.Now()
	if err := uc.downloadProject(ctx, logger, task.ProjectID, tempDir); err != nil {
		return nil, err
	}
	metrics.StageDuration.WithLabelValues("download").Observe(time.Since(downloadStart).Seconds())

	blendPath := filepath.Join(tempDir, extractRelativePath(task.ProjectID, task.MainFileURL))
	outputDir := filepath.Join(tempDir, "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output dir: %w", err)
	}

	job := uc.prepareJob(task, blendPath, outputDir)

	// Setup real-time progress reporting via fsnotify
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Warn("failed to create fsnotify watcher, falling back to no-progress mode", slog.Any("error", err))
	} else {
		defer watcher.Close()
		if err := watcher.Add(outputDir); err != nil {
			logger.Warn("failed to add output directory to watcher", slog.Any("error", err))
		}
	}

	done := make(chan struct{})
	currentFrame := task.StartFrame

	if watcher != nil {
		go func() {
			for {
				select {
				case event, ok := <-watcher.Events:
					if !ok {
						return
					}
					// When a new file is created in the output directory, it's a new frame
					if event.Has(fsnotify.Create) {
						entries, _ := os.ReadDir(outputDir)
						// Filter for common image/video extensions to avoid counting temp files
						count := 0
						for _, entry := range entries {
							if !entry.IsDir() {
								ext := strings.ToLower(filepath.Ext(entry.Name()))
								if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".exr" || ext == ".mp4" || ext == ".tga" || ext == ".bmp" {
									count++
									metrics.FramesRendered.Inc()
								}
							}
						}

						if count > 0 {
							// Update progress based on actual files on disk
							currentFrame = task.StartFrame + count - 1

							reportErr := uc.resultProducer.ReportResult(ctx, domain.RenderResult{
								TaskID:       task.TaskID,
								ProjectID:    task.ProjectID,
								Type:         domain.TypeRender,
								Status:       domain.StatusProcessing,
								WorkerID:     uc.getWorkerID(),
								CurrentFrame: currentFrame,
							})
							if reportErr != nil {
								logger.Warn("failed to report progress", slog.Any("error", reportErr))
							}
						}
					}
				case <-done:
					return
				}
			}
		}()
	}

	renderStart := time.Now()
	renderOutput, err := uc.renderer.Render(ctx, logger, job)
	close(done)
	if err != nil {
		uc.reportError(ctx, logger, task, err, renderOutput)
		return renderOutput, err
	}
	metrics.StageDuration.WithLabelValues("render").Observe(time.Since(renderStart).Seconds())

	uploadStart := time.Now()
	if err := uc.uploadResults(ctx, logger, task.ProjectID, outputDir); err != nil {
		uc.reportError(ctx, logger, task, err, renderOutput)
		return renderOutput, err
	}
	metrics.StageDuration.WithLabelValues("upload").Observe(time.Since(uploadStart).Seconds())

	err = uc.resultProducer.ReportResult(ctx, domain.RenderResult{
		TaskID:    task.TaskID,
		ProjectID: task.ProjectID,
		Type:      domain.TypeRender,
		Status:    domain.StatusCompleted,
		WorkerID:  uc.getWorkerID(),
	})
	if err != nil {
		logger.Error("failed to report result", slog.Any("error", err))
	}

	return renderOutput, nil
}

func (uc *RenderUseCase) reportError(ctx context.Context, logger *slog.Logger, task domain.RenderTask, err error, logs []byte) {
	reportErr := uc.resultProducer.ReportResult(ctx, domain.RenderResult{
		TaskID:    task.TaskID,
		ProjectID: task.ProjectID,
		Type:      domain.TypeRender,
		Status:    domain.StatusFailed,
		WorkerID:  uc.getWorkerID(),
		Error:     err.Error(),
		Logs:      truncateLogs(string(logs), 50000), // Keep last 50KB
	})
	if reportErr != nil {
		logger.Error("failed to report failure result", slog.Any("error", reportErr))
	}
}

func truncateLogs(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return "...[TRUNCATED]...\n" + s[len(s)-maxLen:]
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
func extractRelativePath(projectID, mainFileURL string) string {
	prefix := fmt.Sprintf("%s_sources/", projectID)
	return strings.TrimPrefix(mainFileURL, prefix)
}

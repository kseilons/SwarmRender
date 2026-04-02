package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"kseilons/stitching-service/internall/domain"
)

type ProjectRepository interface {
	DownloadFrames(ctx context.Context, projectID string, destDir string) error
	UploadFinalVideo(ctx context.Context, projectID string, filePath string) (string, error)
}

type ResultProducer interface {
	ReportResult(ctx context.Context, result domain.RenderResult) error
}

type Stitcher interface {
	Stitch(ctx context.Context, logger *slog.Logger, framesDir string, outputFile string, frames []string) error
}

type StitchingUseCase struct {
	projectRepo    ProjectRepository
	resultProducer ResultProducer
	stitcher       Stitcher
}

func NewStitchingUseCase(projectRepo ProjectRepository, resultProducer ResultProducer, stitcher Stitcher) *StitchingUseCase {
	return &StitchingUseCase{
		projectRepo:    projectRepo,
		resultProducer: resultProducer,
		stitcher:       stitcher,
	}
}

func (uc *StitchingUseCase) Execute(ctx context.Context, logger *slog.Logger, task domain.StitchingTask) error {
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("stitch-%s-*", task.ProjectID))
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	logger.Info("downloading frames for stitching", slog.String("project_id", task.ProjectID))
	if err := uc.projectRepo.DownloadFrames(ctx, task.ProjectID, tempDir); err != nil {
		uc.reportError(ctx, logger, task, err)
		return fmt.Errorf("failed to download frames: %w", err)
	}

	// Identify frame pattern and format
	frames, err := uc.getFrames(tempDir)
	if err != nil || len(frames) == 0 {
		uc.reportError(ctx, logger, task, fmt.Errorf("no frames found for stitching"))
		return fmt.Errorf("no frames found")
	}

	outputFile := filepath.Join(tempDir, "output.mp4")

	// Run FFmpeg via Stitcher
	logger.Info("starting stitching", slog.Int("frame_count", len(frames)))
	if err := uc.stitcher.Stitch(ctx, logger, tempDir, outputFile, frames); err != nil {
		uc.reportError(ctx, logger, task, err)
		return err
	}

	// Upload Result
	resultPath, err := uc.projectRepo.UploadFinalVideo(ctx, task.ProjectID, outputFile)
	if err != nil {
		uc.reportError(ctx, logger, task, err)
		return err
	}

	// Report Success
	err = uc.resultProducer.ReportResult(ctx, domain.RenderResult{
		TaskID:    task.TaskID,
		ProjectID: task.ProjectID,
		Type:      domain.TypeStitching,
		Status:    domain.StatusCompleted,
		WorkerID:  os.Getenv("HOSTNAME"),
		ResultURL: resultPath,
	})
	if err != nil {
		logger.Error("failed to report stitching result", slog.Any("error", err))
		return err
	}

	logger.Info("stitching completed successfully", slog.String("result_url", resultPath))
	return nil
}

func (uc *StitchingUseCase) getFrames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var frames []string
	for _, e := range entries {
		if !e.IsDir() && (strings.HasSuffix(e.Name(), ".png") || strings.HasSuffix(e.Name(), ".jpg")) {
			frames = append(frames, e.Name())
		}
	}
	sort.Strings(frames)
	return frames, nil
}

func (uc *StitchingUseCase) reportError(ctx context.Context, logger *slog.Logger, task domain.StitchingTask, err error) {
	reportErr := uc.resultProducer.ReportResult(ctx, domain.RenderResult{
		TaskID:    task.TaskID,
		ProjectID: task.ProjectID,
		Type:      domain.TypeStitching,
		Status:    domain.StatusFailed,
		WorkerID:  os.Getenv("HOSTNAME"),
		Error:     err.Error(),
	})
	if reportErr != nil {
		logger.Error("failed to report failure result", slog.Any("error", reportErr))
	}
}

// FFmpegStitcher implements Stitcher interface using ffmpeg command line
type FFmpegStitcher struct{}

func NewFFmpegStitcher() *FFmpegStitcher {
	return &FFmpegStitcher{}
}

func (s *FFmpegStitcher) Stitch(ctx context.Context, logger *slog.Logger, framesDir string, outputFile string, frames []string) error {
	concatFile := filepath.Join(framesDir, "frames.txt")
	if err := s.createConcatFile(concatFile, frames); err != nil {
		return err
	}

	// ffmpeg -f concat -safe 0 -i frames.txt -c:v libx264 -pix_fmt yuv420p output.mp4
	cmd := exec.CommandContext(ctx, "ffmpeg", "-y", "-f", "concat", "-safe", "0", "-i", concatFile, "-c:v", "libx264", "-pix_fmt", "yuv420p", outputFile)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg failed: %w, output: %s", err, string(output))
	}

	return nil
}

func (s *FFmpegStitcher) createConcatFile(path string, frames []string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, frame := range frames {
		fmt.Fprintf(f, "file '%s'\n", frame)
		fmt.Fprintf(f, "duration 0.04166666666\n")
	}
	return nil
}

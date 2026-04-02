package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"kseilons/job-service/internall/config"
	"kseilons/job-service/internall/domain"

	"github.com/google/uuid"
)

type JobRepository interface {
	CreateJob(ctx context.Context, job domain.Job) error
	GetJob(ctx context.Context, id string) (domain.Job, error)
	UpdateJobStatus(ctx context.Context, id string, status domain.Status) error
	UpdateJobResult(ctx context.Context, id string, status domain.Status, resultPath *string) error
	UpdateJobPreview(ctx context.Context, id string, previewPath string) error
	UpdateJobMetadata(ctx context.Context, id string, frameStart, frameEnd int) error
	UpdateJobProgress(ctx context.Context, id string, completedFrames int) error
	ListJobs(ctx context.Context) ([]domain.Job, error)
}

type TaskRepository interface {
	CreateTasks(ctx context.Context, tasks []domain.Task) error
	GetTask(ctx context.Context, id string) (domain.Task, error)
	GetTasksByJobID(ctx context.Context, jobID string) ([]domain.Task, error)
	UpdateTaskStatus(ctx context.Context, id string, status domain.Status, workerID *string, resultPath *string) (string, error)
	UpdateTaskProgress(ctx context.Context, id string, currentFrame int) (string, error)
	GetJobTasksStatus(ctx context.Context, jobID string) (map[domain.Status]int, error)
	ResetFailedTasks(ctx context.Context, jobID string) ([]domain.Task, error)
}

type TaskProducer interface {
	SendTask(ctx context.Context, projectID string, task any, topic string) error
	Close() error
}

type MinioRepository interface {
	GenerateUploadURL(ctx context.Context, objectKey string) (string, error)
	GenerateDownloadURL(ctx context.Context, objectKey string) (string, error)
	CopyObject(ctx context.Context, srcKey, dstKey string) error
}

type JobUseCase struct {
	jobRepo   JobRepository
	taskRepo  TaskRepository
	minioRepo MinioRepository
	producer  TaskProducer
	kafkaCfg  config.KafkaConfig
	hub       *Hub
}

func NewJobUseCase(jobRepo JobRepository, taskRepo TaskRepository, minioRepo MinioRepository, producer TaskProducer, cfg config.KafkaConfig, hub *Hub) *JobUseCase {
	return &JobUseCase{
		jobRepo:   jobRepo,
		taskRepo:  taskRepo,
		minioRepo: minioRepo,
		producer:  producer,
		kafkaCfg:  cfg,
		hub:       hub,
	}
}

func (uc *JobUseCase) SetKafkaConfig(cfg config.KafkaConfig) {
	uc.kafkaCfg = cfg
}

func (uc *JobUseCase) CreateJob(ctx context.Context, req domain.CreateJobRequest, logger *slog.Logger) (domain.Job, error) {
	job := domain.Job{
		ID:          uuid.New().String(),
		Name:        req.Name,
		S3Path:      req.S3Path,
		FrameStart:  req.FrameStart,
		FrameEnd:    req.FrameEnd,
		WorkerCount: req.WorkerCount,
		Options:     req.Options,
		PreviewURL:  req.PreviewURL,
		Status:      domain.StatusCreated,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	logger = logger.With(job.LogAttrs()...)
	logger.Info("creating new job")

	if err := uc.jobRepo.CreateJob(ctx, job); err != nil {
		logger.Error("failed to create job in repo", slog.Any("error", err))
		return domain.Job{}, fmt.Errorf("failed to create job: %w", err)
	}

	// 2. Decide if we need metadata first
	if job.FrameStart == 0 && job.FrameEnd == 0 {
		logger.Info("missing frame range, requesting metadata")
		if err := uc.RequestMetadata(ctx, job, logger); err != nil {
			return job, err
		}
		return job, nil
	}

	// 3. Otherwise, split and send render tasks
	if err := uc.SplitJobIntoTasks(ctx, job, logger); err != nil {
		return job, err
	}

	logger.Info("job created and tasks dispatched successfully")
	return job, nil
}

func (uc *JobUseCase) RequestMetadata(ctx context.Context, job domain.Job, logger *slog.Logger) error {
	task := domain.MetadataTask{
		BaseTask: domain.BaseTask{
			ProjectID: extractProjectID(job.S3Path),
			TaskID:    fmt.Sprintf("meta-%s", job.ID),
		},
		BlendFile: job.S3Path,
	}

	topic := uc.kafkaCfg.MetadataTopic
	if topic == "" {
		topic = "metadata-tasks"
	}

	logger.Info("sending metadata task",
		slog.String("task_id", task.TaskID),
		slog.String("topic", topic),
	)

	if err := uc.producer.SendTask(ctx, task.ProjectID, task, topic); err != nil {
		logger.Error("failed to send metadata task",
			slog.String("topic", topic),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to send metadata task: %w", err)
	}

	return uc.UpdateJobStatus(ctx, job.ID, domain.StatusPendingMetadata, logger)
}

func (uc *JobUseCase) SplitJobIntoTasks(ctx context.Context, job domain.Job, logger *slog.Logger) error {
	logger.Info("splitting job into tasks", slog.Int("worker_count", job.WorkerCount))

	totalFrames := job.FrameEnd - job.FrameStart + 1
	if totalFrames <= 0 {
		return fmt.Errorf("invalid frame range: %d to %d", job.FrameStart, job.FrameEnd)
	}

	framesPerWorker := totalFrames / job.WorkerCount
	if framesPerWorker == 0 {
		framesPerWorker = 1
	}

	var tasks []domain.Task
	currentFrame := job.FrameStart

	for i := 0; i < job.WorkerCount && currentFrame <= job.FrameEnd; i++ {
		endFrame := currentFrame + framesPerWorker - 1
		if i == job.WorkerCount-1 || endFrame > job.FrameEnd {
			endFrame = job.FrameEnd
		}

		task := domain.Task{
			ID:         uuid.New().String(),
			JobID:      job.ID,
			Type:       domain.TypeRender,
			FrameStart: currentFrame,
			FrameEnd:   endFrame,
			Status:     domain.StatusCreated,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		tasks = append(tasks, task)
		currentFrame = endFrame + 1
	}

	if err := uc.taskRepo.CreateTasks(ctx, tasks); err != nil {
		return fmt.Errorf("failed to save tasks: %w", err)
	}

	// Send to Kafka
	topic := uc.kafkaCfg.RenderTopic
	for _, t := range tasks {
		renderTask := domain.RenderTask{
			BaseTask: domain.BaseTask{
				ProjectID: extractProjectID(job.S3Path),
				TaskID:    t.ID,
			},
			CommonTask: domain.CommonTask{
				StartFrame: t.FrameStart,
				EndFrame:   t.FrameEnd,
				Options:    job.Options,
			},
			BlendFile: job.S3Path,
		}

		if err := uc.producer.SendTask(ctx, job.ID, renderTask, topic); err != nil {
			logger.Error("failed to send render task", slog.String("task_id", t.ID), slog.Any("error", err))
		}
	}

	return uc.UpdateJobStatus(ctx, job.ID, domain.StatusProcessing, logger)
}

func (uc *JobUseCase) HandleMetadataResult(ctx context.Context, jobID string, frameStart, frameEnd int, logger *slog.Logger) error {
	logger.Info("handling metadata result",
		slog.String("job_id", jobID),
		slog.Int("f_start", frameStart),
		slog.Int("f_end", frameEnd),
	)

	if err := uc.jobRepo.UpdateJobMetadata(ctx, jobID, frameStart, frameEnd); err != nil {
		return fmt.Errorf("failed to update job metadata: %w", err)
	}

	job, err := uc.jobRepo.GetJob(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job after metadata update: %w", err)
	}

	return uc.SplitJobIntoTasks(ctx, job, logger)
}

func (uc *JobUseCase) HandleTaskResult(ctx context.Context, result domain.RenderResult, logger *slog.Logger) error {
	logger.Info("handling task result", slog.String("task_id", result.TaskID), slog.String("status", string(result.Status)))

	jobID, err := uc.taskRepo.UpdateTaskStatus(ctx, result.TaskID, result.Status, result.WorkerID, result.ResultPath)
	if err != nil {
		logger.Error("failed to update task status", slog.Any("error", err))
		return err
	}

	if result.Status == domain.StatusCompleted {
		task, err := uc.taskRepo.GetTask(ctx, result.TaskID)
		if err == nil {
			uc.HandleTaskProgress(ctx, result.TaskID, task.FrameEnd, logger)
		}
	}

	if uc.hub != nil {
		uc.hub.BroadcastStatusUpdate(map[string]any{
			"type":    "task_update",
			"task_id": result.TaskID,
			"status":  result.Status,
		})
	}

	// Check job status
	stats, err := uc.taskRepo.GetJobTasksStatus(ctx, jobID)
	if err != nil {
		logger.Error("failed to get job tasks status", slog.String("job_id", jobID), slog.Any("error", err))
		return nil // Don't fail the whole result handling
	}

	totalTasks := 0
	for _, count := range stats {
		totalTasks += count
	}

	if stats[domain.StatusCompleted] == totalTasks {
		logger.Info("all tasks completed, transitioning job to STITCHING", slog.String("job_id", jobID))
		job, err := uc.jobRepo.GetJob(ctx, jobID)
		if err != nil {
			logger.Error("failed to get job for stitching", slog.String("job_id", jobID), slog.Any("error", err))
			return nil
		}
		if err := uc.UpdateJobStatus(ctx, jobID, domain.StatusStitching, logger); err != nil {
			return nil
		}
		return uc.StartStitching(ctx, job, logger)
	}

	if stats[domain.StatusFailed] == totalTasks {
		logger.Info("all tasks failed, transitioning job to FAILED", slog.String("job_id", jobID))
		return uc.UpdateJobStatus(ctx, jobID, domain.StatusFailed, logger)
	}

	// Preview generation logic
	if result.Status == domain.StatusCompleted && result.Type == domain.TypeRender {
		job, err := uc.jobRepo.GetJob(ctx, jobID)
		if err == nil && (job.PreviewURL == nil || *job.PreviewURL == "") {
			task, err := uc.taskRepo.GetTask(ctx, result.TaskID)
			if err == nil {
				uc.generatePreview(ctx, job, task, logger)
			}
		}
	}

	// If some tasks failed but not all, we might want to mark job as FAILED eventually,
	// but for now we follow the user's "if all ERROR -> FAILED"
	if stats[domain.StatusFailed] > 0 && (stats[domain.StatusCompleted]+stats[domain.StatusFailed] == totalTasks) {
		logger.Info("some tasks failed, transitioning job to FAILED", slog.String("job_id", jobID))
		return uc.UpdateJobStatus(ctx, jobID, domain.StatusFailed, logger)
	}

	return nil
}

func (uc *JobUseCase) StartStitching(ctx context.Context, job domain.Job, logger *slog.Logger) error {
	stitchingTaskID := uuid.New().String()

	dbTask := domain.Task{
		ID:         stitchingTaskID,
		JobID:      job.ID,
		Type:       domain.TypeStitching,
		FrameStart: job.FrameStart, // Use job frames for reference
		FrameEnd:   job.FrameEnd,
		Status:     domain.StatusStitching,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := uc.taskRepo.CreateTasks(ctx, []domain.Task{dbTask}); err != nil {
		logger.Error("failed to create stitching task record", slog.Any("error", err))
	}

	task := domain.StitchingTask{
		BaseTask: domain.BaseTask{
			ProjectID: extractProjectID(job.S3Path),
			TaskID:    stitchingTaskID,
		},
		S3Path:     job.S3Path,
		FrameStart: job.FrameStart,
		FrameEnd:   job.FrameEnd,
		Format:     job.Options.Format,
	}

	topic := uc.kafkaCfg.StitchingTopic
	if topic == "" {
		topic = "stitching-tasks"
	}

	logger.Info("sending stitching task",
		slog.String("task_id", task.TaskID),
		slog.String("topic", topic),
	)

	if err := uc.producer.SendTask(ctx, task.ProjectID, task, topic); err != nil {
		logger.Error("failed to send stitching task",
			slog.String("topic", topic),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to send stitching task: %w", err)
	}

	return nil
}

func (uc *JobUseCase) HandleTaskProgress(ctx context.Context, taskID string, currentFrame int, logger *slog.Logger) error {
	jobID, err := uc.taskRepo.UpdateTaskProgress(ctx, taskID, currentFrame)
	if err != nil {
		return err
	}

	if uc.hub != nil {
		uc.hub.BroadcastStatusUpdate(map[string]any{
			"type":          "task_progress",
			"task_id":       taskID,
			"current_frame": currentFrame,
		})
	}

	// Calculate and update job progress
	tasks, err := uc.taskRepo.GetTasksByJobID(ctx, jobID)
	if err == nil {
		totalCompleted := 0
		for _, t := range tasks {
			if t.Status == domain.StatusCompleted {
				totalCompleted += (t.FrameEnd - t.FrameStart + 1)
			} else if t.CurrentFrame >= t.FrameStart {
				totalCompleted += (t.CurrentFrame - t.FrameStart + 1)
			}
		}
		_ = uc.jobRepo.UpdateJobProgress(ctx, jobID, totalCompleted)

		if uc.hub != nil {
			uc.hub.BroadcastStatusUpdate(map[string]any{
				"type":             "job_progress",
				"job_id":           jobID,
				"completed_frames": totalCompleted,
			})
		}
	}

	return nil
}

func (uc *JobUseCase) HandleStitchingResult(ctx context.Context, result domain.RenderResult, logger *slog.Logger) error {
	logger.Info("handling stitching result",
		slog.String("task_id", result.TaskID),
		slog.String("status", string(result.Status)),
		slog.Any("result_path", result.ResultPath),
	)

	jobID, err := uc.taskRepo.UpdateTaskStatus(ctx, result.TaskID, result.Status, result.WorkerID, result.ResultPath)
	if err != nil {
		logger.Error("failed to update stitching task in tasks table", slog.Any("error", err))
	}

	if uc.hub != nil {
		uc.hub.BroadcastStatusUpdate(map[string]any{
			"type":    "task_update",
			"task_id": result.TaskID,
			"status":  result.Status,
		})
	}

	if jobID == "" {
		jobID = result.ProjectID
	}

	err = uc.jobRepo.UpdateJobResult(ctx, jobID, result.Status, result.ResultPath)
	if err == nil && uc.hub != nil {
		uc.hub.BroadcastStatusUpdate(map[string]any{
			"type":        "job_update",
			"job_id":      jobID,
			"status":      result.Status,
			"result_path": result.ResultPath,
		})
	}
	return err
}

func (uc *JobUseCase) UpdateJobStatus(ctx context.Context, id string, status domain.Status, logger *slog.Logger) error {
	logger.Info("updating job status",
		slog.String("job_id", id),
		slog.String("new_status", string(status)),
	)

	if err := uc.jobRepo.UpdateJobStatus(ctx, id, status); err != nil {
		logger.Error("failed to update job status",
			slog.String("job_id", id),
			slog.Any("error", err),
		)
		return err
	}

	if uc.hub != nil {
		uc.hub.BroadcastStatusUpdate(map[string]any{
			"type":   "job_update",
			"job_id": id,
			"status": status,
		})
	}
	return nil
}

func (uc *JobUseCase) GetJob(ctx context.Context, id string) (domain.JobResponse, error) {
	job, err := uc.jobRepo.GetJob(ctx, id)
	if err != nil {
		return domain.JobResponse{}, err
	}

	tasks, err := uc.taskRepo.GetTasksByJobID(ctx, id)
	if err != nil {
		return domain.JobResponse{}, err
	}

	if job.PreviewURL != nil && *job.PreviewURL != "" && !strings.HasPrefix(*job.PreviewURL, "http") {
		url, err := uc.minioRepo.GenerateDownloadURL(ctx, *job.PreviewURL)
		if err == nil {
			job.PreviewURL = &url
		}
	}

	if job.ResultPath != nil && *job.ResultPath != "" && strings.HasSuffix(*job.ResultPath, ".mp4") {
		// If it's a video path from stitching service, it might be an s3:// URL or a key
		key := strings.TrimPrefix(*job.ResultPath, "s3://")
		// Remove bucket name from key if present
		parts := strings.SplitN(key, "/", 2)
		if len(parts) == 2 {
			key = parts[1]
		}
		url, err := uc.minioRepo.GenerateDownloadURL(ctx, key)
		if err == nil {
			job.ResultPath = &url
		}
	}

	return domain.JobResponse{
		Job:   job,
		Tasks: tasks,
	}, nil
}

func (uc *JobUseCase) generatePreview(ctx context.Context, job domain.Job, task domain.Task, logger *slog.Logger) {
	if uc.minioRepo == nil {
		return
	}
	projectID := extractProjectID(job.S3Path)
	scenePrefix := ""
	if job.Options.Scene != "" {
		scenePrefix = fmt.Sprintf("%s_", job.Options.Scene)
	}

	frameNum := task.FrameStart
	srcKey := fmt.Sprintf("%s_results/%s%04d.png", projectID, scenePrefix, frameNum)
	dstKey := fmt.Sprintf("%s_preview.png", projectID)

	logger.Info("generating preview", slog.String("src", srcKey), slog.String("dst", dstKey))

	if err := uc.minioRepo.CopyObject(ctx, srcKey, dstKey); err != nil {
		logger.Error("failed to copy preview object", slog.Any("error", err))
		return
	}

	if err := uc.jobRepo.UpdateJobPreview(ctx, job.ID, dstKey); err != nil {
		logger.Error("failed to update job preview path", slog.Any("error", err))
	}
}

func (uc *JobUseCase) ListJobs(ctx context.Context) ([]domain.Job, error) {
	jobs, err := uc.jobRepo.ListJobs(ctx)
	if err != nil {
		return nil, err
	}

	for i := range jobs {
		if jobs[i].PreviewURL != nil && *jobs[i].PreviewURL != "" && !strings.HasPrefix(*jobs[i].PreviewURL, "http") {
			url, err := uc.minioRepo.GenerateDownloadURL(ctx, *jobs[i].PreviewURL)
			if err == nil {
				jobs[i].PreviewURL = &url
			}
		}
		if jobs[i].ResultPath != nil && *jobs[i].ResultPath != "" && !strings.HasPrefix(*jobs[i].ResultPath, "http") {
			key := strings.TrimPrefix(*jobs[i].ResultPath, "s3://")
			// Remove bucket name from key if it's a full s3:// URL
			parts := strings.SplitN(key, "/", 2)
			if len(parts) == 2 {
				key = parts[1]
			}
			url, err := uc.minioRepo.GenerateDownloadURL(ctx, key)
			if err == nil {
				jobs[i].ResultPath = &url
			}
		}
	}
	return jobs, nil
}

func (uc *JobUseCase) RetryJob(ctx context.Context, jobID string, logger *slog.Logger) error {
	logger.Info("retrying failed tasks for job", slog.String("job_id", jobID))

	job, err := uc.jobRepo.GetJob(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	resetTasks, err := uc.taskRepo.ResetFailedTasks(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to reset failed tasks: %w", err)
	}

	if len(resetTasks) == 0 {
		logger.Info("no failed tasks to retry", slog.String("job_id", jobID))
		return nil
	}

	// Send to Kafka
	topic := uc.kafkaCfg.RenderTopic
	for _, t := range resetTasks {
		renderTask := domain.RenderTask{
			BaseTask: domain.BaseTask{
				ProjectID: extractProjectID(job.S3Path),
				TaskID:    t.ID,
			},
			CommonTask: domain.CommonTask{
				StartFrame: t.FrameStart,
				EndFrame:   t.FrameEnd,
				Options:    job.Options,
			},
			BlendFile: job.S3Path,
		}

		if err := uc.producer.SendTask(ctx, job.ID, renderTask, topic); err != nil {
			logger.Error("failed to re-send render task", slog.String("task_id", t.ID), slog.Any("error", err))
		}
	}

	return uc.UpdateJobStatus(ctx, job.ID, domain.StatusProcessing, logger)
}

func (uc *JobUseCase) GenerateUploadURL(ctx context.Context, filename string, projectID string) (string, string, error) {
	if projectID == "" {
		projectID = uuid.New().String()
	}
	objectKey := fmt.Sprintf("%s_sources/%s", projectID, filename)

	url, err := uc.minioRepo.GenerateUploadURL(ctx, objectKey)
	if err != nil {
		return "", "", err
	}

	return url, objectKey, nil
}

func extractProjectID(s3Path string) string {
	parts := strings.Split(s3Path, "_sources/")
	if len(parts) > 0 {
		return parts[0]
	}
	return s3Path
}

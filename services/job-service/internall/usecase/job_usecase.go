package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"kseilons/job-service/internall/config"
	"kseilons/job-service/internall/domain"

	"github.com/google/uuid"
)

type JobRepository interface {
	CreateJob(ctx context.Context, job domain.Job) error
	GetJob(ctx context.Context, id string) (domain.Job, error)
	UpdateJobStatus(ctx context.Context, id string, status domain.Status) error
	UpdateJobMetadata(ctx context.Context, id string, frameStart, frameEnd int) error
	ListJobs(ctx context.Context) ([]domain.Job, error)
}

type TaskRepository interface {
	CreateTasks(ctx context.Context, tasks []domain.Task) error
	GetTasksByJobID(ctx context.Context, jobID string) ([]domain.Task, error)
	UpdateTaskStatus(ctx context.Context, id string, status domain.Status, workerID string, resultPath string) (string, error)
	GetJobTasksStatus(ctx context.Context, jobID string) (map[domain.Status]int, error)
}

type TaskProducer interface {
	SendTask(ctx context.Context, projectID string, task any, topic string) error
	Close() error
}

type JobUseCase struct {
	jobRepo  JobRepository
	taskRepo TaskRepository
	producer TaskProducer
	kafkaCfg config.KafkaConfig
	hub      *Hub
}

func NewJobUseCase(jobRepo JobRepository, taskRepo TaskRepository, producer TaskProducer, cfg config.KafkaConfig, hub *Hub) *JobUseCase {
	return &JobUseCase{
		jobRepo:  jobRepo,
		taskRepo: taskRepo,
		producer: producer,
		kafkaCfg: cfg,
		hub:      hub,
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
			ProjectID: job.ID,
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
				ProjectID: job.ID,
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

	// If some tasks failed but not all, we might want to mark job as FAILED eventually, 
	// but for now we follow the user's "if all ERROR -> FAILED"
	if stats[domain.StatusFailed] > 0 && (stats[domain.StatusCompleted]+stats[domain.StatusFailed] == totalTasks) {
		logger.Info("some tasks failed, transitioning job to FAILED", slog.String("job_id", jobID))
		return uc.UpdateJobStatus(ctx, jobID, domain.StatusFailed, logger)
	}

	return nil
}

func (uc *JobUseCase) StartStitching(ctx context.Context, job domain.Job, logger *slog.Logger) error {
	task := domain.StitchingTask{
		BaseTask: domain.BaseTask{
			ProjectID: job.ID,
			TaskID:    fmt.Sprintf("stitch-%s", job.ID),
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

	return domain.JobResponse{
		Job:   job,
		Tasks: tasks,
	}, nil
}

func (uc *JobUseCase) ListJobs(ctx context.Context) ([]domain.Job, error) {
	return uc.jobRepo.ListJobs(ctx)
}

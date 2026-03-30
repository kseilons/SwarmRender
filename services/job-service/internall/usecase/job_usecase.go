package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"kseilons/job-service/internall/config"
	"kseilons/job-service/internall/domain"
)

type JobRepository interface {
	CreateJob(ctx context.Context, job domain.Job) error
	GetJob(ctx context.Context, id string) (domain.Job, error)
	UpdateJobStatus(ctx context.Context, id string, status domain.Status) error
}

type TaskProducer interface {
	SendTask(ctx context.Context, projectID string, task any, topic string) error
	Close() error
}

type JobUseCase struct {
	repo     JobRepository
	producer TaskProducer
	kafkaCfg config.KafkaConfig
}

func NewJobUseCase(repo JobRepository, producer TaskProducer, cfg config.KafkaConfig) *JobUseCase {
	return &JobUseCase{
		repo:     repo,
		producer: producer,
		kafkaCfg: cfg,
	}
}

func (uc *JobUseCase) SetKafkaConfig(cfg config.KafkaConfig) {
	uc.kafkaCfg = cfg
}

func (uc *JobUseCase) CreateJob(ctx context.Context, job domain.Job, logger *slog.Logger) error {
	logger = logger.With(job.LogAttrs()...)
	logger.Info("creating new job")

	// 1. Initial job record
	job.Status = domain.StatusCreated
	if err := uc.repo.CreateJob(ctx, job); err != nil {
		logger.Error("failed to create job in repo", slog.Any("error", err))
		return fmt.Errorf("failed to create job: %w", err)
	}

	// 2. Decide if we need metadata first
	if job.FrameStart == 0 && job.FrameEnd == 0 {
		logger.Info("missing frame range, requesting metadata")
		return uc.RequestMetadata(ctx, job, logger)
	}

	// 3. Otherwise, split and send render tasks (simplified for now)
	logger.Info("job created successfully")
	return nil
}

func (uc *JobUseCase) RequestMetadata(ctx context.Context, job domain.Job, logger *slog.Logger) error {
	task := domain.MetadataTask{
		BaseTask: domain.BaseTask{
			ProjectID: job.ID,
			TaskID:    fmt.Sprintf("meta-%s", job.ID),
		},
		BlendFile: "project.blend", // Should come from job or s3_path
	}

	topic := "metadata-tasks" // Fallback
	if uc.kafkaCfg.MetadataTopic != "" {
		topic = uc.kafkaCfg.MetadataTopic
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

func (uc *JobUseCase) UpdateJobStatus(ctx context.Context, id string, status domain.Status, logger *slog.Logger) error {
	logger.Info("updating job status",
		slog.String("job_id", id),
		slog.String("new_status", string(status)),
	)

	if err := uc.repo.UpdateJobStatus(ctx, id, status); err != nil {
		logger.Error("failed to update job status",
			slog.String("job_id", id),
			slog.Any("error", err),
		)
		return err
	}
	return nil
}

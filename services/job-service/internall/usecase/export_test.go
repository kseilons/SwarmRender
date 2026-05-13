package usecase

import (
	"context"
	"kseilons/job-service/internall/domain"
	"log/slog"
)

func (uc *JobUseCase) ExportGeneratePreview(ctx context.Context, job domain.Job, task domain.Task, logger *slog.Logger) {
	uc.generatePreview(ctx, job, task, logger)
}

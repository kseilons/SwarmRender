package domain

import "log/slog"

type Status string

const (
	StatusPending   Status = "PENDING"
	StatusCompleted Status = "COMPLETED"
	StatusFailed    Status = "FAILED"
)

type TaskType string

const (
	TypeMetadata  TaskType = "METADATA"
	TypeRender    TaskType = "RENDER"
	TypeStitching TaskType = "STITCHING"
)

type BaseTask struct {
	ProjectID string `json:"project_id"`
	TaskID    string `json:"task_id"`
}

type StitchingTask struct {
	BaseTask
	S3Path     string `json:"s3_path"`
	FrameStart int    `json:"frame_start"`
	FrameEnd   int    `json:"frame_end"`
	Format     string `json:"format"`
}

type RenderResult struct {
	TaskID     string   `json:"task_id"`
	ProjectID  string   `json:"project_id"`
	Type       TaskType `json:"type"`
	Status     Status   `json:"status"`
	WorkerID   string   `json:"worker_id"`
	Error      string   `json:"error,omitempty"`
	ResultPath string   `json:"result_path,omitempty"`
}

func (t StitchingTask) LogAttrs() []any {
	return []any{
		slog.String("project_id", t.ProjectID),
		slog.String("task_id", t.TaskID),
		slog.String("s3_path", t.S3Path),
		slog.Int("f_start", t.FrameStart),
		slog.Int("f_end", t.FrameEnd),
	}
}

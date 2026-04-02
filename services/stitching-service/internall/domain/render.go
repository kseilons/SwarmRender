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
	MainFileURL string `json:"main_file_url"`
	FrameStart  int    `json:"frame_start"`
	FrameEnd    int    `json:"frame_end"`
	Format      string `json:"format"`
}

type RenderResult struct {
	TaskID    string   `json:"task_id"`
	ProjectID string   `json:"project_id"`
	Type      TaskType `json:"type"`
	Status    Status   `json:"status"`
	WorkerID  string   `json:"worker_id"`
	Error     string   `json:"error,omitempty"`
	ResultURL string   `json:"result_url,omitempty"`
}

func (t StitchingTask) LogAttrs() []any {
	return []any{
		slog.String("project_id", t.ProjectID),
		slog.String("task_id", t.TaskID),
		slog.String("main_file_url", t.MainFileURL),
		slog.Int("f_start", t.FrameStart),
		slog.Int("f_end", t.FrameEnd),
	}
}

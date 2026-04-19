package domain

import (
	"log/slog"
	"time"
)

type Status string

const (
	StatusCreated         Status = "CREATED"
	StatusPendingMetadata Status = "PENDING_METADATA"
	StatusProcessing      Status = "PROCESSING"
	StatusCompleted       Status = "COMPLETED"
	StatusFailed          Status = "FAILED"
	StatusStitching       Status = "STITCHING"
)

type AuthUser struct {
	ID    string   `json:"id"`
	Email string   `json:"email"`
	Roles []string `json:"roles"`
}

type RenderOptions struct {
	Engine       string `json:"engine"`
	UseCPU       bool   `json:"use_cpu"`
	CyclesDevice string `json:"cycles_device"`
	Threads      int    `json:"threads"`
	Format       string `json:"format"`
	Scene        string `json:"scene"`
	Background   bool   `json:"background"`
}

type CommonTask struct {
	StartFrame int           `json:"start_frame"`
	EndFrame   int           `json:"end_frame"`
	Options    RenderOptions `json:"options"`
}

type BaseTask struct {
	ProjectID string    `json:"project_id"`
	TaskID    string    `json:"task_id"`
	CreatedAt time.Time `json:"created_at"`
}

type MetadataTask struct {
	BaseTask
	MainFileURL string `json:"main_file_url"`
}

type RenderTask struct {
	BaseTask
	CommonTask  `json:",inline"`
	MainFileURL string `json:"main_file_url"`
}

type Job struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	MainFileURL     string        `json:"main_file_url"`
	FrameStart      int           `json:"frame_start"`
	FrameEnd        int           `json:"frame_end"`
	WorkerCount     int           `json:"worker_count"`
	Status          Status        `json:"status"`
	ResultURL       *string       `json:"result_url,omitempty"`
	Options         RenderOptions `json:"options"`
	PreviewURL      *string       `json:"preview_url,omitempty"`
	CompletedFrames int           `json:"completed_frames"`
	OwnerID         string        `json:"owner_id,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

type Task struct {
	ID           string    `json:"id"`
	JobID        string    `json:"job_id"`
	Type         TaskType  `json:"type"`
	FrameStart   int       `json:"frame_start"`
	FrameEnd     int       `json:"frame_end"`
	Status       Status    `json:"status"`
	WorkerID     *string   `json:"worker_id,omitempty"`
	ResultURL    *string   `json:"result_url,omitempty"`
	RetryCount   int       `json:"retry_count"`
	CurrentFrame int       `json:"current_frame"`
	Logs         *string   `json:"logs,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateJobRequest struct {
	Name        string        `json:"name" binding:"required"`
	MainFileURL string        `json:"main_file_url" binding:"required"`
	FrameStart  int           `json:"frame_start"`
	FrameEnd    int           `json:"frame_end"`
	WorkerCount int           `json:"worker_count" binding:"required"`
	Options     RenderOptions `json:"options"`
	PreviewURL  *string       `json:"preview_url"`
}

type JobResponse struct {
	Job   Job    `json:"job"`
	Tasks []Task `json:"tasks,omitempty"`
}

type ProjectMetadata struct {
	FrameStart int `json:"frame_start"`
	FrameEnd   int `json:"frame_end"`
}

type TaskType string

const (
	TypeMetadata  TaskType = "METADATA"
	TypeRender    TaskType = "RENDER"
	TypeStitching TaskType = "STITCHING"
)

type RenderResult struct {
	TaskID       string           `json:"task_id"`
	ProjectID    string           `json:"project_id"`
	Type         TaskType         `json:"type"`
	Status       Status           `json:"status"`
	WorkerID     *string          `json:"worker_id"`
	Error        string           `json:"error,omitempty"`
	Logs         string           `json:"logs,omitempty"`
	Metadata     *ProjectMetadata `json:"metadata,omitempty"`
	ResultURL    *string          `json:"result_url,omitempty"`
	CurrentFrame int              `json:"current_frame,omitempty"`
}

type ClusterStats struct {
	TotalWorkers int `json:"total_workers"`
}

type StitchingTask struct {
	BaseTask
	MainFileURL string `json:"main_file_url"`
	FrameStart  int    `json:"frame_start"`
	FrameEnd    int    `json:"frame_end"`
	Format      string `json:"format"`
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

func (j Job) LogAttrs() []any {
	return []any{
		slog.String("job_id", j.ID),
		slog.String("name", j.Name),
		slog.String("status", string(j.Status)),
		slog.Any("preview_url", j.PreviewURL),
		slog.Int("f_start", j.FrameStart),
		slog.Int("f_end", j.FrameEnd),
	}
}

func (t Task) LogAttrs() []any {
	return []any{
		slog.String("task_id", t.ID),
		slog.String("job_id", t.JobID),
		slog.String("status", string(t.Status)),
	}
}

func (t MetadataTask) LogAttrs() []any {
	return []any{
		slog.String("project_id", t.ProjectID),
		slog.String("task_id", t.TaskID),
		slog.String("main_file_url", t.MainFileURL),
	}
}

func (t RenderTask) LogAttrs() []any {
	return []any{
		slog.String("project_id", t.ProjectID),
		slog.String("task_id", t.TaskID),
		slog.String("main_file_url", t.MainFileURL),
		slog.Int("f_start", t.StartFrame),
		slog.Int("f_end", t.EndFrame),
	}
}

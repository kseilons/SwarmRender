package domain

import "log/slog"

// EngineType defines the available render engines in Blender.
type EngineType string

const (
	EngineCycles    EngineType = "CYCLES"
	EngineEeveeNext EngineType = "BLENDER_EEVEE_NEXT"
	EngineWorkbench EngineType = "BLENDER_WORKBENCH"
)

// CyclesDevice defines the device used by the Cycles engine.
type CyclesDevice string

const (
	DeviceCPU    CyclesDevice = "CPU"
	DeviceCUDA   CyclesDevice = "CUDA"
	DeviceOptix  CyclesDevice = "OPTIX"
	DeviceHip    CyclesDevice = "HIP"
	DeviceOneAPI CyclesDevice = "ONEAPI"
	DeviceMetal  CyclesDevice = "METAL"
)

// FileFormat defines the output image/video format.
type FileFormat string

const (
	FormatPNG     FileFormat = "PNG"
	FormatJPEG    FileFormat = "JPEG"
	FormatTGA     FileFormat = "TGA"
	FormatBMP     FileFormat = "BMP"
	FormatHDR     FileFormat = "HDR"
	FormatTIFF    FileFormat = "TIFF"
	FormatOpenEXR FileFormat = "OPEN_EXR"
	FormatFFMPEG  FileFormat = "FFMPEG"
)

type Status string

const (
	StatusPending   Status = "PENDING"
	StatusCompleted Status = "COMPLETED"
	StatusFailed    Status = "FAILED"
)



type ProjectMetadata struct {
	FrameStart int `json:"frame_start"`
	FrameEnd   int `json:"frame_end"`
}

// RenderOptions contains configuration for a render task.
type RenderOptions struct {
	Engine       EngineType   `json:"engine"`
	UseCPU       bool         `json:"use_cpu"`
	CyclesDevice CyclesDevice `json:"cycles_device"`
	Threads      int          `json:"threads"` // 0 to use all system processors
	Format       FileFormat   `json:"format"`
	Scene        string       `json:"scene"`      // Optional: specific scene to render
	Background   bool         `json:"background"` // Usually true for worker services
}

type CommonTask struct {
	StartFrame int           `json:"start_frame"`
	EndFrame   int           `json:"end_frame"`
	Options    RenderOptions `json:"options"`
}

// BaseTask contains common fields for all tasks sent over Kafka.
type BaseTask struct {
	ProjectID string `json:"project_id"`
	TaskID    string `json:"task_id"`
}

// MetadataTask contains parameters for a metadata extraction job.
type MetadataTask struct {
	BaseTask
	BlendFile string `json:"blend_file"`
}

// RenderTask groups all parameters for a single render job.
type RenderTask struct {
	BaseTask
	CommonTask `json:",inline"`
	BlendFile  string `json:"blend_file"`
}

type TaskType string

const (
	TypeMetadata  TaskType = "METADATA"
	TypeRender    TaskType = "RENDER"
	TypeStitching TaskType = "STITCHING"
)

// RenderResult is the message workers send back when a task is finished.
type RenderResult struct {
	TaskID     string           `json:"task_id"`
	ProjectID  string           `json:"project_id"`
	Type       TaskType         `json:"type"`
	Status     Status           `json:"status"`
	WorkerID   string           `json:"worker_id"`
	Error      string           `json:"error,omitempty"`
	Metadata   *ProjectMetadata `json:"metadata,omitempty"`
	ResultPath string           `json:"result_path,omitempty"`
}


// RenderJob contains all parameters needed by the rendering engine.
type RenderJob struct {
	CommonTask
	BlendPath  string
	OutputPath string
}

// LogAttrs returns a slice of slog attributes for the metadata task.
func (t MetadataTask) LogAttrs() []any {
	return []any{
		slog.String("project_id", t.ProjectID),
		slog.String("blend_file", t.BlendFile),
	}
}

// LogAttrs returns a slice of slog attributes for the render task.
func (t RenderTask) LogAttrs() []any {
	return []any{
		slog.String("project_id", t.ProjectID),
		slog.String("blend_file", t.BlendFile),
		slog.Int("start_frame", t.StartFrame),
		slog.Int("end_frame", t.EndFrame),
		slog.String("engine", string(t.Options.Engine)),
	}
}

// LogAttrs returns a slice of slog attributes for the internal job.
func (j RenderJob) LogAttrs() []any {
	return []any{
		slog.String("blend_path", j.BlendPath),
		slog.String("output_path", j.OutputPath),
		slog.Int("start_frame", j.StartFrame),
		slog.Int("end_frame", j.EndFrame),
		slog.String("engine", string(j.Options.Engine)),
	}
}

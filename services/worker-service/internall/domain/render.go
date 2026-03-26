package domain

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

// RenderOptions contains configuration for a render task.
type RenderOptions struct {
	Engine       EngineType
	UseCPU       bool
	CyclesDevice CyclesDevice
	Threads      int // 0 to use all system processors
	Format       FileFormat
	Scene        string // Optional: specific scene to render
	Background   bool   // Usually true for worker services
}

// RenderTask groups all parameters for a single render job.
type RenderTask struct {
	BlendPath  string
	StartFrame int
	EndFrame   int
	OutputPath string
	Options    RenderOptions
}

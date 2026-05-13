package render

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"kseilons/worker-service/internall/domain"
)

func TestEngine_BuildArgs(t *testing.T) {
	e := NewEngine("/usr/bin/blender")

	tests := []struct {
		name     string
		job      domain.RenderJob
		expected []string
	}{
		{
			name: "background render with scene and format",
			job: domain.RenderJob{
				CommonTask: domain.CommonTask{
					Options: domain.RenderOptions{
						Background: true,
						Engine:     domain.EngineCycles,
						Format:     domain.FormatPNG,
						Scene:      "MainScene",
					},
				},
				BlendPath:  "/tmp/test.blend",
				OutputPath: "/tmp/output/",
			},
			expected: []string{"-b", "/tmp/test.blend", "-E", "CYCLES", "-o", "/tmp/output/", "-F", "PNG", "-S", "MainScene", "--"},
		},
		{
			name: "single frame render",
			job: domain.RenderJob{
				CommonTask: domain.CommonTask{
					StartFrame: 10,
					EndFrame:   10,
				},
				BlendPath: "/tmp/test.blend",
			},
			expected: []string{"/tmp/test.blend", "-f", "10"},
		},
		{
			name: "frame range render",
			job: domain.RenderJob{
				CommonTask: domain.CommonTask{
					StartFrame: 1,
					EndFrame:   10,
				},
				BlendPath: "/tmp/test.blend",
			},
			expected: []string{"/tmp/test.blend", "-s", "1", "-e", "10", "-a"},
		},
		{
			name: "cycles with CUDA device",
			job: domain.RenderJob{
				CommonTask: domain.CommonTask{
					Options: domain.RenderOptions{
						Engine:       domain.EngineCycles,
						CyclesDevice: domain.DeviceCUDA,
					},
				},
				BlendPath: "/tmp/test.blend",
			},
			expected: []string{"/tmp/test.blend", "-E", "CYCLES", "--", "--cycles-device", "CUDA"},
		},
		{
			name: "cycles with CUDA+CPU",
			job: domain.RenderJob{
				CommonTask: domain.CommonTask{
					Options: domain.RenderOptions{
						Engine:       domain.EngineCycles,
						CyclesDevice: domain.DeviceCUDA,
						UseCPU:       true,
					},
				},
				BlendPath: "/tmp/test.blend",
			},
			expected: []string{"/tmp/test.blend", "-E", "CYCLES", "--", "--cycles-device", "CUDA+CPU"},
		},
		{
			name: "threads and scene",
			job: domain.RenderJob{
				CommonTask: domain.CommonTask{
					Options: domain.RenderOptions{
						Threads: 4,
						Scene:   "Final",
					},
				},
				BlendPath: "/tmp/test.blend",
			},
			expected: []string{"/tmp/test.blend", "-t", "4", "-S", "Final"},
		},
		{
			name: "cycles with CPU only (no +CPU suffix)",
			job: domain.RenderJob{
				CommonTask: domain.CommonTask{
					Options: domain.RenderOptions{
						Engine:       domain.EngineCycles,
						CyclesDevice: domain.DeviceCPU,
						UseCPU:       true,
					},
				},
				BlendPath: "/tmp/test.blend",
			},
			expected: []string{"/tmp/test.blend", "-E", "CYCLES", "--", "--cycles-device", "CPU"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := e.BuildArgs(tt.job)
			assert.Equal(t, tt.expected, args)
		})
	}
}

func TestEngine_Render_ValidationFailure(t *testing.T) {
	e := NewEngine("blender")
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	job := domain.RenderJob{
		BlendPath: "", // trigger error
	}

	output, err := e.Render(context.Background(), logger, job)
	assert.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "blend file path is required")
}

func TestEngine_Render_ExecutionFailure(t *testing.T) {
	// Use a non-existent mobile blender executable to trigger failure
	e := NewEngine("/path/to/non-existent/blender")
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	job := domain.RenderJob{
		BlendPath: "/tmp/test.blend",
	}

	_, err := e.Render(context.Background(), logger, job)
	assert.Error(t, err)
	// If executable is not found, output is usually nil in this case
}

func TestEngine_GetMetadata_Success(t *testing.T) {
	// Create a fake blender script
	wd, _ := os.Getwd()
	scriptPath := filepath.Join(wd, "fake_blender.bat")
	scriptContent := "@echo off\necho Some blender noise...\necho FRAME_INFO:{\"frame_start\": 10, \"frame_end\": 200}\n"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	assert.NoError(t, err)
	defer os.Remove(scriptPath)

	e := NewEngine(scriptPath)
	ctx := context.Background()

	meta, err := e.GetMetadata(ctx, "test.blend")
	assert.NoError(t, err)
	assert.Equal(t, 10, meta.FrameStart)
	assert.Equal(t, 200, meta.FrameEnd)
}

func TestEngine_GetMetadata_ParseError(t *testing.T) {
	wd, _ := os.Getwd()
	scriptPath := filepath.Join(wd, "fake_blender_fail.bat")
	scriptContent := "@echo off\necho FRAME_INFO:invalid-json\n"
	os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	defer os.Remove(scriptPath)

	e := NewEngine(scriptPath)
	_, err := e.GetMetadata(context.Background(), "test.blend")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse metadata JSON")
}

func TestEngine_GetMetadata_NoInfo(t *testing.T) {
	wd, _ := os.Getwd()
	scriptPath := filepath.Join(wd, "fake_blender_empty.bat")
	scriptContent := "@echo off\necho No info here\n"
	os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	defer os.Remove(scriptPath)

	e := NewEngine(scriptPath)
	_, err := e.GetMetadata(context.Background(), "test.blend")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not find FRAME_INFO")
}

func TestEngine_Render_Success(t *testing.T) {
	wd, _ := os.Getwd()
	scriptPath := filepath.Join(wd, "fake_render.bat")
	scriptContent := "@echo off\necho Blender 4.0.2\necho Fra:1 Mem:120M ^| Remaining:00:10.50\necho Rendered frame 1\n"
	os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	defer os.Remove(scriptPath)

	e := NewEngine(scriptPath)
	logger := slog.Default()
	job := domain.RenderJob{BlendPath: "test.blend"}

	output, err := e.Render(context.Background(), logger, job)
	assert.NoError(t, err)
	assert.Contains(t, string(output), "Fra:1")
	assert.Contains(t, string(output), "Rendered frame 1")
}

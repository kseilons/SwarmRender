package render

import (
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := e.BuildArgs(tt.job)
			assert.Equal(t, tt.expected, args)
		})
	}
}

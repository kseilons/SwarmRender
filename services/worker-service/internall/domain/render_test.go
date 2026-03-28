package domain

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderTask_LogAttrs(t *testing.T) {
	task := RenderTask{
		ProjectID: "proj-123",
		BlendFile: "main.blend",
		CommonTask: CommonTask{
			StartFrame: 1,
			EndFrame:   10,
			Options: RenderOptions{
				Engine: EngineCycles,
			},
		},
	}

	attrs := task.LogAttrs()

	// Convert slice of any to map for easier assertion
	attrMap := make(map[string]any)
	for i := 0; i < len(attrs); i++ {
		if attr, ok := attrs[i].(slog.Attr); ok {
			attrMap[attr.Key] = attr.Value.Any()
		}
	}

	assert.Equal(t, "proj-123", attrMap["project_id"])
	assert.Equal(t, "main.blend", attrMap["blend_file"])
	assert.Equal(t, int64(1), attrMap["start_frame"])
	assert.Equal(t, int64(10), attrMap["end_frame"])
	assert.Equal(t, "CYCLES", attrMap["engine"])
}

func TestRenderJob_LogAttrs(t *testing.T) {
	job := RenderJob{
		BlendPath:  "/tmp/scene.blend",
		OutputPath: "/tmp/output",
		CommonTask: CommonTask{
			StartFrame: 1,
			EndFrame:   1,
			Options: RenderOptions{
				Engine: EngineEeveeNext,
			},
		},
	}

	attrs := job.LogAttrs()

	attrMap := make(map[string]any)
	for i := 0; i < len(attrs); i++ {
		if attr, ok := attrs[i].(slog.Attr); ok {
			attrMap[attr.Key] = attr.Value.Any()
		}
	}

	assert.Equal(t, "/tmp/scene.blend", attrMap["blend_path"])
	assert.Equal(t, "/tmp/output", attrMap["output_path"])
	assert.Equal(t, "BLENDER_EEVEE_NEXT", attrMap["engine"])
}

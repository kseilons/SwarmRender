package domain_test

import (
	"kseilons/stitching-service/internall/domain"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStitchingTask_LogAttrs(t *testing.T) {
	task := domain.StitchingTask{
		BaseTask: domain.BaseTask{
			ProjectID: "proj-1",
			TaskID:    "task-1",
		},
		MainFileURL: "http://example.com/file",
		FrameStart:  0,
		FrameEnd:    100,
	}

	attrs := task.LogAttrs()
	assert.Len(t, attrs, 5)
}

package usecase

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFFmpegStitcher_createConcatFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "stitch-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	s := &FFmpegStitcher{}
	path := filepath.Join(tempDir, "frames.txt")
	frames := []string{"frame1.png", "frame2.jpg"}

	err = s.createConcatFile(path, frames)
	assert.NoError(t, err)

	content, err := os.ReadFile(path)
	assert.NoError(t, err)

	expected := "file 'frame1.png'\nduration 0.04166666666\nfile 'frame2.jpg'\nduration 0.04166666666\n"
	assert.Equal(t, expected, string(content))
}

func TestStitchingUseCase_getFrames(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "frames-test-*")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		os.WriteFile(filepath.Join(tempDir, "001.png"), []byte("data"), 0644)
		os.WriteFile(filepath.Join(tempDir, "003.jpg"), []byte("data"), 0644)
		os.WriteFile(filepath.Join(tempDir, "readme.txt"), []byte("data"), 0644)
		os.Mkdir(filepath.Join(tempDir, "subdir"), 0755)

		uc := &StitchingUseCase{}
		frames, err := uc.getFrames(tempDir)
		assert.NoError(t, err)
		assert.Equal(t, []string{"001.png", "003.jpg"}, frames)
	})

	t.Run("Error", func(t *testing.T) {
		uc := &StitchingUseCase{}
		_, err := uc.getFrames("/non-existent-path-hopefully")
		assert.Error(t, err)
	})
}

func TestNewStitchingUseCase(t *testing.T) {
	uc := NewStitchingUseCase(nil, nil, nil)
	assert.NotNil(t, uc)
}

func TestNewFFmpegStitcher(t *testing.T) {
	s := NewFFmpegStitcher()
	assert.NotNil(t, s)
}

func TestFFmpegStitcher_Stitch_Fail(t *testing.T) {
	// Testing with non-existent ffmpeg
	s := &FFmpegStitcher{}
	err := s.Stitch(context.Background(), slog.Default(), "dir", "out", []string{"f1.png"})
	assert.Error(t, err)
}

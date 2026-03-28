package repository

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"kseilons/worker-service/internall/repository/mocks"
)

func TestMinioRepository_DownloadProject_Errors(t *testing.T) {
	bucket := "test-bucket"
	projectID := "proj-1"

	t.Run("ListObjectsError", func(t *testing.T) {
		mockClient := mocks.NewClient(t)
		repo := &MinioRepository{client: mockClient, bucket: bucket}

		ch := make(chan minio.ObjectInfo, 1)
		ch <- minio.ObjectInfo{Err: errors.New("list error")}
		close(ch)

		mockClient.On("ListObjects", mock.Anything, bucket, mock.Anything).Return((<-chan minio.ObjectInfo)(ch))

		err := repo.DownloadProject(context.Background(), projectID, "/tmp")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list objects")
	})

	t.Run("FGetObjectError", func(t *testing.T) {
		mockClient := mocks.NewClient(t)
		repo := &MinioRepository{client: mockClient, bucket: bucket}

		ch := make(chan minio.ObjectInfo, 1)
		ch <- minio.ObjectInfo{Key: "proj-1_sources/scene.blend"}
		close(ch)

		mockClient.On("ListObjects", mock.Anything, bucket, mock.Anything).Return((<-chan minio.ObjectInfo)(ch))
		mockClient.On("FGetObject", mock.Anything, bucket, "proj-1_sources/scene.blend", mock.Anything, mock.Anything).
			Return(errors.New("download error"))

		destDir, _ := os.MkdirTemp("", "unit-test-*")
		defer os.RemoveAll(destDir)

		err := repo.DownloadProject(context.Background(), projectID, destDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to download object")
	})
}

func TestMinioRepository_UploadResult_Errors(t *testing.T) {
	bucket := "test-bucket"
	projectID := "proj-1"

	t.Run("FPutObjectError", func(t *testing.T) {
		mockClient := mocks.NewClient(t)
		repo := &MinioRepository{client: mockClient, bucket: bucket}

		resDir, _ := os.MkdirTemp("", "unit-res-*")
		defer os.RemoveAll(resDir)
		_ = os.WriteFile(filepath.Join(resDir, "res.png"), []byte("data"), 0644)

		mockClient.On("FPutObject", mock.Anything, bucket, "proj-1_results/res.png", mock.Anything, mock.Anything).
			Return(minio.UploadInfo{}, errors.New("upload error"))

		err := repo.UploadResult(context.Background(), projectID, resDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to upload object")
	})
}

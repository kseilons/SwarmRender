package repository

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"kseilons/worker-service/internall/config"
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

func TestMinioRepository_Success(t *testing.T) {
	bucket := "test-bucket"
	projectID := "proj-1"
	ctx := context.Background()

	t.Run("DownloadProject_Success", func(t *testing.T) {
		mockClient := mocks.NewClient(t)
		repo := &MinioRepository{client: mockClient, bucket: bucket}

		ch := make(chan minio.ObjectInfo, 2)
		ch <- minio.ObjectInfo{Key: "proj-1_sources/a.blend"}
		ch <- minio.ObjectInfo{Key: "proj-1_sources/subdir/b.png"}
		close(ch)

		mockClient.On("ListObjects", mock.Anything, bucket, mock.Anything).Return((<-chan minio.ObjectInfo)(ch))
		mockClient.On("FGetObject", mock.Anything, bucket, mock.Anything, mock.Anything, mock.Anything).Return(nil).Twice()

		destDir, _ := os.MkdirTemp("", "unit-down-*")
		defer os.RemoveAll(destDir)

		err := repo.DownloadProject(ctx, projectID, destDir)
		assert.NoError(t, err)
	})

	t.Run("UploadResult_Success", func(t *testing.T) {
		mockClient := mocks.NewClient(t)
		repo := &MinioRepository{client: mockClient, bucket: bucket}

		resDir, _ := os.MkdirTemp("", "unit-up-*")
		defer os.RemoveAll(resDir)
		_ = os.MkdirAll(filepath.Join(resDir, "nested"), 0755)
		_ = os.WriteFile(filepath.Join(resDir, "f1.png"), []byte("d1"), 0644)
		_ = os.WriteFile(filepath.Join(resDir, "nested/f2.png"), []byte("d2"), 0644)

		mockClient.On("FPutObject", mock.Anything, bucket, mock.MatchedBy(func(s string) bool { return strings.HasPrefix(s, "proj-1_results/") }), mock.Anything, mock.Anything).
			Return(minio.UploadInfo{}, nil).Twice()

		err := repo.UploadResult(ctx, projectID, resDir)
		assert.NoError(t, err)
	})
}

func TestNewMinioRepository(t *testing.T) {
	cfg := config.MinioConfig{
		Endpoint:  "localhost:9000",
		AccessKey: "admin",
		SecretKey: "password",
		Bucket:    "test",
	}
	repo, err := NewMinioRepository(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, repo)
}

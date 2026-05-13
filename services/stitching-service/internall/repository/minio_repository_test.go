package repository

import (
	"context"
	"errors"
	"testing"

	"kseilons/stitching-service/internall/config"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockMinioClient struct {
	mock.Mock
}

func (m *MockMinioClient) ListObjects(ctx context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
	args := m.Called(ctx, bucketName, opts)
	return args.Get(0).(<-chan minio.ObjectInfo)
}

func (m *MockMinioClient) FGetObject(ctx context.Context, bucketName, objectName, filePath string, opts minio.GetObjectOptions) error {
	args := m.Called(ctx, bucketName, objectName, filePath, opts)
	return args.Error(0)
}

func (m *MockMinioClient) FPutObject(ctx context.Context, bucketName, objectName, filePath string, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	args := m.Called(ctx, bucketName, objectName, filePath, opts)
	return args.Get(0).(minio.UploadInfo), args.Error(1)
}

func TestMinioRepository_DownloadFrames_Success(t *testing.T) {
	mockClient := new(MockMinioClient)
	repo := &MinioRepository{
		client: mockClient,
		bucket: "test-bucket",
	}

	ctx := context.Background()
	objectCh := make(chan minio.ObjectInfo, 4)
	objectCh <- minio.ObjectInfo{Key: "proj-1_results/"} // Должен пропустить (префикс)
	objectCh <- minio.ObjectInfo{Key: "proj-1_results/readme.txt"} // Должен пропустить (не картинка)
	objectCh <- minio.ObjectInfo{Key: "proj-1_results/frame1.png"}
	close(objectCh)

	mockClient.On("ListObjects", ctx, "test-bucket", mock.Anything).Return((<-chan minio.ObjectInfo)(objectCh)).Once()
	mockClient.On("FGetObject", ctx, "test-bucket", "proj-1_results/frame1.png", mock.Anything, mock.Anything).Return(nil).Once()

	err := repo.DownloadFrames(ctx, "proj-1", "tmp")
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestMinioRepository_DownloadFrames_FGetObjectError(t *testing.T) {
	mockClient := new(MockMinioClient)
	repo := &MinioRepository{
		client: mockClient,
		bucket: "test-bucket",
	}

	ctx := context.Background()
	objectCh := make(chan minio.ObjectInfo, 1)
	objectCh <- minio.ObjectInfo{Key: "proj-1_results/frame1.png"}
	close(objectCh)

	mockClient.On("ListObjects", ctx, "test-bucket", mock.Anything).Return((<-chan minio.ObjectInfo)(objectCh)).Once()
	mockClient.On("FGetObject", ctx, "test-bucket", "proj-1_results/frame1.png", mock.Anything, mock.Anything).Return(errors.New("download failed")).Once()

	err := repo.DownloadFrames(ctx, "proj-1", "tmp")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to download frame")
}

func TestMinioRepository_DownloadFrames_ListError(t *testing.T) {
	mockClient := new(MockMinioClient)
	repo := &MinioRepository{
		client: mockClient,
		bucket: "test-bucket",
	}

	ctx := context.Background()
	objectCh := make(chan minio.ObjectInfo, 1)
	objectCh <- minio.ObjectInfo{Err: errors.New("minio error")}
	close(objectCh)

	mockClient.On("ListObjects", ctx, "test-bucket", mock.Anything).Return((<-chan minio.ObjectInfo)(objectCh)).Once()

	err := repo.DownloadFrames(ctx, "proj-1", "tmp")
	assert.Error(t, err)
}

func TestMinioRepository_UploadFinalVideo_Success(t *testing.T) {
	mockClient := new(MockMinioClient)
	repo := &MinioRepository{
		client:       mockClient,
		outputBucket: "output-bucket",
	}

	ctx := context.Background()
	mockClient.On("FPutObject", ctx, "output-bucket", "proj-1_final.mp4", "file.mp4", mock.Anything).Return(minio.UploadInfo{}, nil).Once()

	name, err := repo.UploadFinalVideo(ctx, "proj-1", "file.mp4")
	assert.NoError(t, err)
	assert.Equal(t, "proj-1_final.mp4", name)
}

func TestMinioRepository_UploadFinalVideo_Error(t *testing.T) {
	mockClient := new(MockMinioClient)
	repo := &MinioRepository{
		client:       mockClient,
		outputBucket: "output-bucket",
	}

	ctx := context.Background()
	mockClient.On("FPutObject", ctx, "output-bucket", "proj-1_final.mp4", "file.mp4", mock.Anything).Return(minio.UploadInfo{}, errors.New("upload error")).Once()

	_, err := repo.UploadFinalVideo(ctx, "proj-1", "file.mp4")
	assert.Error(t, err)
}

func TestNewMinioRepository(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		cfg := config.MinioConfig{
			Endpoint:  "localhost:9000",
			AccessKey: "access",
			SecretKey: "secret",
		}
		repo, err := NewMinioRepository(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, repo)
	})

	t.Run("InvalidEndpoint", func(t *testing.T) {
		cfg := config.MinioConfig{
			Endpoint: "invalid space",
		}
		_, err := NewMinioRepository(cfg)
		assert.Error(t, err)
	})
}

package repository

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"kseilons/stitching-service/internall/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Client interface {
	ListObjects(ctx context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo
	FGetObject(ctx context.Context, bucketName, objectName, filePath string, opts minio.GetObjectOptions) error
	FPutObject(ctx context.Context, bucketName, objectName, filePath string, opts minio.PutObjectOptions) (minio.UploadInfo, error)
}

type MinioRepository struct {
	client       Client
	bucket       string
	outputBucket string
}

func NewMinioRepository(cfg config.MinioConfig) (*MinioRepository, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	return &MinioRepository{
		client:       client,
		bucket:       cfg.Bucket,
		outputBucket: cfg.OutputBucket,
	}, nil
}

func (r *MinioRepository) DownloadFrames(ctx context.Context, projectID string, destDir string) error {
	prefix := fmt.Sprintf("%s_results/", projectID)
	objectCh := r.client.ListObjects(ctx, r.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			return fmt.Errorf("failed to list objects: %w", object.Err)
		}
		if object.Key == prefix {
			continue
		}

		if !strings.HasSuffix(object.Key, ".png") && !strings.HasSuffix(object.Key, ".jpg") {
			continue
		}

		destPath := filepath.Join(destDir, filepath.Base(object.Key))
		err := r.client.FGetObject(ctx, r.bucket, object.Key, destPath, minio.GetObjectOptions{})
		if err != nil {
			return fmt.Errorf("failed to download frame %s: %w", object.Key, err)
		}
	}

	return nil
}

func (r *MinioRepository) UploadFinalVideo(ctx context.Context, projectID string, filePath string) (string, error) {
	objectName := fmt.Sprintf("%s_final.mp4", projectID)
	_, err := r.client.FPutObject(ctx, r.outputBucket, objectName, filePath, minio.PutObjectOptions{
		ContentType: "video/mp4",
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload final video: %w", err)
	}

	return objectName, nil
}

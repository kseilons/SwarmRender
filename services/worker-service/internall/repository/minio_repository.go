package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"kseilons/worker-service/internall/config"
)

type MinioRepository struct {
	client *minio.Client
	bucket string
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
		client: client,
		bucket: cfg.Bucket,
	}, nil
}

// DownloadProject downloads all objects with prefix {projectID}_sources/ to destDir.
func (r *MinioRepository) DownloadProject(ctx context.Context, projectID string, destDir string) error {
	prefix := fmt.Sprintf("%s_sources/", projectID)
	
	// List objects with prefix
	objectCh := r.client.ListObjects(ctx, r.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			return fmt.Errorf("failed to list objects: %w", object.Err)
		}

		// Relative path from prefix
		relPath := strings.TrimPrefix(object.Key, prefix)
		if relPath == "" {
			continue
		}

		localPath := filepath.Join(destDir, relPath)

		// Create directory if it doesn't exist
		if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		// Download object
		err := r.client.FGetObject(ctx, r.bucket, object.Key, localPath, minio.GetObjectOptions{})
		if err != nil {
			return fmt.Errorf("failed to download object %s: %w", object.Key, err)
		}
	}

	return nil
}

// UploadResult uploads all files in resultDir to MinIO with prefix {projectID}_results/.
func (r *MinioRepository) UploadResult(ctx context.Context, projectID string, resultDir string) error {
	prefix := fmt.Sprintf("%s_results/", projectID)

	err := filepath.Walk(resultDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Relative path for MinIO key
		relPath, err := filepath.Rel(resultDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		objectKey := filepath.ToSlash(filepath.Join(prefix, relPath))

		_, err = r.client.FPutObject(ctx, r.bucket, objectKey, path, minio.PutObjectOptions{})
		if err != nil {
			return fmt.Errorf("failed to upload object %s: %w", objectKey, err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk result directory: %w", err)
	}

	return nil
}

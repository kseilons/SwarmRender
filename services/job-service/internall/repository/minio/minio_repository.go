package minio

import (
	"context"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"kseilons/job-service/internall/config"
)

type MinioRepository struct {
	client *minio.Client
	bucket string
	expiry time.Duration
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
		expiry: time.Duration(cfg.PresignedExpiry) * time.Second,
	}, nil
}

func (r *MinioRepository) GenerateUploadURL(ctx context.Context, objectKey string) (string, error) {
	presignedURL, err := r.client.PresignedPutObject(ctx, r.bucket, objectKey, r.expiry)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned upload url: %w", err)
	}

	return presignedURL.String(), nil
}

func (r *MinioRepository) GenerateDownloadURL(ctx context.Context, objectKey string) (string, error) {
	presignedURL, err := r.client.PresignedGetObject(ctx, r.bucket, objectKey, r.expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned download url: %w", err)
	}

	return presignedURL.String(), nil
}

func (r *MinioRepository) CopyObject(ctx context.Context, srcKey, dstKey string) error {
	src := minio.CopySrcOptions{
		Bucket: r.bucket,
		Object: srcKey,
	}

	dst := minio.CopyDestOptions{
		Bucket: r.bucket,
		Object: dstKey,
	}

	_, err := r.client.CopyObject(ctx, dst, src)
	if err != nil {
		return fmt.Errorf("failed to copy object from %s to %s: %w", srcKey, dstKey, err)
	}

	return nil
}

//go:build integration

package repository_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"kseilons/worker-service/internall/config"
	"kseilons/worker-service/internall/repository"
)

func TestMinioRepository_Integration(t *testing.T) {
	// 1. Setup config from file
	allCfg, err := config.Load("../../config/config-test.yaml")
	if err != nil {
		t.Fatalf("failed to load test config: %v", err)
	}
	cfg := allCfg.Minio

	ctx := context.Background()

	// Initialize real MinIO client to prepare state
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		t.Fatalf("failed to connect to minio: %v (make sure docker-compose is up)", err)
	}

	// Create bucket
	err = client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{})
	if err != nil {
		exists, errBucketExists := client.BucketExists(ctx, cfg.Bucket)
		if errBucketExists == nil && exists {
			// Bucket already exists, it's fine
		} else {
			t.Fatalf("failed to create bucket: %v", err)
		}
	}
	defer func() {
		// Cleanup: Remove all objects and bucket
		objectsCh := client.ListObjects(ctx, cfg.Bucket, minio.ListObjectsOptions{Recursive: true})
		for obj := range objectsCh {
			_ = client.RemoveObject(ctx, cfg.Bucket, obj.Key, minio.RemoveObjectOptions{})
		}
		_ = client.RemoveBucket(ctx, cfg.Bucket)
	}()

	repo, err := repository.NewMinioRepository(cfg)
	assert.NoError(t, err)

	projectID := "test-proj-123"

	t.Run("DownloadProject", func(t *testing.T) {
		// 1. Upload dummy sources
		files := map[string]string{
			"scene.blend":       "dummy content",
			"textures/wood.png": "pixel data",
		}

		for name, content := range files {
			key := fmt.Sprintf("%s_sources/%s", projectID, name)
			tmpFile, _ := os.CreateTemp("", "source-*")
			_, _ = tmpFile.WriteString(content)
			tmpFile.Close()
			defer os.Remove(tmpFile.Name())

			_, err = client.FPutObject(ctx, cfg.Bucket, key, tmpFile.Name(), minio.PutObjectOptions{})
			assert.NoError(t, err)
		}

		// 2. Download project
		destDir, _ := os.MkdirTemp("", "dest-*")
		defer os.RemoveAll(destDir)

		err = repo.DownloadProject(ctx, projectID, destDir)
		assert.NoError(t, err)

		// 3. Verify files
		for name, content := range files {
			path := filepath.Join(destDir, name)
			data, err := os.ReadFile(path)
			assert.NoError(t, err)
			assert.Equal(t, content, string(data))
		}
	})

	t.Run("UploadResult", func(t *testing.T) {
		// 1. Create dummy results locally
		resDir, _ := os.MkdirTemp("", "res-*")
		defer os.RemoveAll(resDir)

		results := map[string]string{
			"render_0001.png": "image data 1",
			"logs.txt":        "render log content",
		}

		for name, content := range results {
			path := filepath.Join(resDir, name)
			_ = os.WriteFile(path, []byte(content), 0644)
		}

		// 2. Upload results
		err = repo.UploadResult(ctx, projectID, resDir)
		assert.NoError(t, err)

		// 3. Verify in MinIO
		for name, content := range results {
			key := fmt.Sprintf("%s_results/%s", projectID, name)
			obj, err := client.GetObject(ctx, cfg.Bucket, key, minio.GetObjectOptions{})
			assert.NoError(t, err)

			data, err := io.ReadAll(obj)
			_ = obj.Close()
			assert.NoError(t, err)
			assert.Equal(t, content, string(data))
		}
	})
}

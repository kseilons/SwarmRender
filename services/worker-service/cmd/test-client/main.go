package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/segmentio/kafka-go"
	"kseilons/worker-service/internall/config"
	"kseilons/worker-service/internall/domain"
)

func main() {
	projectID := flag.String("project", "test-project", "Project ID")
	filePath := flag.String("file", "", "Path to .blend file or project directory")
	startFrame := flag.Int("start", 1, "Start frame")
	endFrame := flag.Int("end", 1, "End frame")
	engine := flag.String("engine", "CYCLES", "Render engine (CYCLES, BLENDER_EEVEE_NEXT, BLENDER_WORKBENCH)")
	configPath := flag.String("config", "config/config.yaml", "Path to config file")
	flag.Parse()

	if *filePath == "" {
		log.Fatal("File path is required. Use -file flag.")
	}

	// 1. Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx := context.Background()

	// 2. Upload to MinIO
	minioClient, err := minio.New(cfg.Minio.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.Minio.AccessKey, cfg.Minio.SecretKey, ""),
		Secure: cfg.Minio.UseSSL,
	})
	if err != nil {
		log.Fatalf("Failed to create minio client: %v", err)
	}

	fmt.Printf("Uploading project %s from %s to MinIO...\n", *projectID, *filePath)

	fileInfo, err := os.Stat(*filePath)
	if err != nil {
		log.Fatalf("Failed to stat file: %v", err)
	}

	var blendFile string
	prefix := fmt.Sprintf("%s_sources/", *projectID)

	if fileInfo.IsDir() {
		// Upload directory
		err = filepath.Walk(*filePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			relPath, _ := filepath.Rel(*filePath, path)
			objectKey := filepath.ToSlash(filepath.Join(prefix, relPath))

			if strings.HasSuffix(info.Name(), ".blend") && blendFile == "" {
				blendFile = relPath
			}

			_, err = minioClient.FPutObject(ctx, cfg.Minio.Bucket, objectKey, path, minio.PutObjectOptions{})
			return err
		})
	} else {
		// Upload single file
		blendFile = filepath.Base(*filePath)
		objectKey := prefix + blendFile
		_, err = minioClient.FPutObject(ctx, cfg.Minio.Bucket, objectKey, *filePath, minio.PutObjectOptions{})
	}

	if err != nil {
		log.Fatalf("Failed to upload: %v", err)
	}
	if blendFile == "" {
		log.Fatal("No .blend file found in the provided path")
	}
	fmt.Printf("Upload successful. Primary blend file: %s\n", blendFile)

	// 3. Send Task to Kafka
	task := domain.RenderTask{
		ProjectID: *projectID,
		BlendFile: blendFile,
		CommonTask: domain.CommonTask{
			StartFrame: *startFrame,
			EndFrame:   *endFrame,
			Options: domain.RenderOptions{
				Engine:       domain.EngineType(*engine),
				Format:       domain.FormatPNG,
				Background:   true,
				UseCPU:       true,
				CyclesDevice: domain.DeviceCPU,
			},
		},
	}

	payload, err := json.Marshal(task)
	if err != nil {
		log.Fatalf("Failed to marshal task: %v", err)
	}

	writer := &kafka.Writer{
		Addr:     kafka.TCP(cfg.Kafka.Brokers...),
		Topic:    cfg.Kafka.Topic,
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	err = writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(*projectID),
		Value: payload,
	})
	if err != nil {
		log.Fatalf("Failed to send task to Kafka: %v", err)
	}

	fmt.Printf("Task sent to Kafka topic %s successfully.\n", cfg.Kafka.Topic)
	fmt.Printf("Project: %s, File: %s, Frames: %d-%d\n", *projectID, blendFile, *startFrame, *endFrame)
}

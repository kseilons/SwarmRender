//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"time"

	"kseilons/worker-service/internall/domain"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *WorkerIntegrationTestSuite) Test_MetadataExtraction() {
	t := s.T()
	ctx := context.Background()

	kafkaBrokers, _ := s.env.Kafka.Brokers(ctx)
	
	// 1. Prepare Task
	task := domain.MetadataTask{
		BaseTask: domain.BaseTask{
			ProjectID: "test-proj-meta",
			TaskID:    "test-task-meta",
			CreatedAt: time.Now(),
		},
		MainFileURL: "scene.blend",
	}
	taskBody, _ := json.Marshal(task)

	// 2. Send Task to Kafka
	writer := &kafka.Writer{
		Addr:     kafka.TCP(kafkaBrokers...),
		Topic:    "metadata-tasks",
		Balancer: &kafka.LeastBytes{},
	}
	err := writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(task.TaskID),
		Value: taskBody,
	})
	require.NoError(t, err)
	defer writer.Close()

	// 3. Listen for Results
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: kafkaBrokers,
		Topic:   "render-results",
		GroupID: "test-result-listener-meta",
	})
	defer reader.Close()

	assert.Eventually(t, func() bool {
		m, err := reader.ReadMessage(ctx)
		if err != nil {
			return false
		}
		var res domain.RenderResult
		json.Unmarshal(m.Value, &res)
		
		return res.ProjectID == "test-proj-meta" && res.Type == domain.TypeMetadata
	}, 20*time.Second, 1*time.Second)
}

func (s *WorkerIntegrationTestSuite) Test_Rendering() {
	t := s.T()
	ctx := context.Background()

	kafkaBrokers, _ := s.env.Kafka.Brokers(ctx)

	// 1. Prepare Render Task
	task := domain.RenderTask{
		BaseTask: domain.BaseTask{
			ProjectID: "test-proj-render",
			TaskID:    "test-task-render",
			CreatedAt: time.Now(),
		},
		CommonTask: domain.CommonTask{
			StartFrame: 1,
			EndFrame:   1,
			Options: domain.RenderOptions{
				Background:   true,
				Format:       domain.FormatPNG,
				Engine:       domain.EngineCycles,
				CyclesDevice: domain.DeviceCPU,
			},
		},
		MainFileURL: "scene.blend",
	}
	taskBody, _ := json.Marshal(task)

	// 2. Send Task to Kafka
	writer := &kafka.Writer{
		Addr:     kafka.TCP(kafkaBrokers...),
		Topic:    "render-tasks-cpu",
		Balancer: &kafka.LeastBytes{},
	}
	err := writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(task.TaskID),
		Value: taskBody,
	})
	require.NoError(t, err)
	defer writer.Close()

	// 3. Listen for Results
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: kafkaBrokers,
		Topic:   "render-results",
		GroupID: "test-result-listener-render",
	})
	defer reader.Close()

	assert.Eventually(t, func() bool {
		m, err := reader.ReadMessage(ctx)
		if err != nil {
			return false
		}
		var res domain.RenderResult
		json.Unmarshal(m.Value, &res)

		return res.ProjectID == "test-proj-render" && res.Type == domain.TypeRender && res.Status == domain.StatusCompleted
	}, 20*time.Second, 1*time.Second)
}

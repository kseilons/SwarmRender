//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"kseilons/job-service/internall/domain"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *JobIntegrationTestSuite) Test_KafkaResultProcessing() {
	t := s.T()
	ctx := context.Background()
	token := s.createToken("test-user-kafka")

	// 1. Create a job
	jobReq := domain.CreateJobRequest{
		Name: "Kafka Job", MainFileURL: "f", WorkerCount: 1,
		FrameStart: 1, FrameEnd: 10,
	}
	body, _ := json.Marshal(jobReq)
	req, _ := http.NewRequest("POST", s.baseURL+"/jobs", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	var job domain.Job
	json.NewDecoder(resp.Body).Decode(&job)

	// 1.5 Fetch real task ID from DB
	var taskID uuid.UUID
	err := s.app.GetDB().QueryRow(ctx, "SELECT id FROM tasks WHERE job_id = $1 LIMIT 1", job.ID).Scan(&taskID)
	require.NoError(t, err)

	// 2. Simulate Result
	result := domain.RenderResult{
		ProjectID: job.ID,
		TaskID:    taskID.String(),
		Type:      domain.TypeRender,
		Status:    domain.StatusCompleted,
	}
	resBody, _ := json.Marshal(result)
	s.kafkaWriter.WriteMessages(ctx, kafka.Message{Key: []byte(job.ID), Value: resBody})

	assert.Eventually(t, func() bool {
		req, _ = http.NewRequest("GET", s.baseURL+"/jobs/"+job.ID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, _ = http.DefaultClient.Do(req)
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 500*time.Millisecond)
}

func (s *JobIntegrationTestSuite) Test_KafkaMetadataProcessing() {
	t := s.T()
	ctx := context.Background()
	token := s.createToken("test-user-meta")

	jobReq := domain.CreateJobRequest{
		Name: "Meta Job", MainFileURL: "f", WorkerCount: 1,
	}
	body, _ := json.Marshal(jobReq)
	req, _ := http.NewRequest("POST", s.baseURL+"/jobs", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	var job domain.Job
	json.NewDecoder(resp.Body).Decode(&job)

	result := domain.RenderResult{
		ProjectID: job.ID,
		TaskID:    "meta-" + job.ID,
		Type:      domain.TypeMetadata,
		Status:    domain.StatusCompleted,
		Metadata: &domain.ProjectMetadata{FrameStart: 10, FrameEnd: 50},
	}
	resBody, _ := json.Marshal(result)
	s.kafkaWriter.WriteMessages(ctx, kafka.Message{Key: []byte(job.ID), Value: resBody})

	assert.Eventually(t, func() bool {
		req, _ = http.NewRequest("GET", s.baseURL+"/jobs/"+job.ID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, _ = http.DefaultClient.Do(req)
		var jobRes domain.JobResponse
		json.NewDecoder(resp.Body).Decode(&jobRes)
		return jobRes.Job.FrameStart == 10 && jobRes.Job.FrameEnd == 50
	}, 10*time.Second, 500*time.Millisecond)
}

func (s *JobIntegrationTestSuite) Test_KafkaStitchingProcessing() {
	t := s.T()
	ctx := context.Background()
	token := s.createToken("test-user-stitch")

	jobReq := domain.CreateJobRequest{
		Name: "Stitch Job", MainFileURL: "f", WorkerCount: 1,
		FrameStart: 1, FrameEnd: 10,
	}
	body, _ := json.Marshal(jobReq)
	req, _ := http.NewRequest("POST", s.baseURL+"/jobs", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	var job domain.Job
	json.NewDecoder(resp.Body).Decode(&job)

	// 1.5 Complete render tasks first to trigger stitching
	var renderTaskID uuid.UUID
	err := s.app.GetDB().QueryRow(ctx, "SELECT id FROM tasks WHERE job_id = $1 AND type = $2 LIMIT 1", job.ID, domain.TypeRender).Scan(&renderTaskID)
	require.NoError(t, err)

	renderResult := domain.RenderResult{
		ProjectID: job.ID,
		TaskID:    renderTaskID.String(),
		Type:      domain.TypeRender,
		Status:    domain.StatusCompleted,
	}
	resBody, _ := json.Marshal(renderResult)
	s.kafkaWriter.WriteMessages(ctx, kafka.Message{Key: []byte(job.ID), Value: resBody})

	// 1.6 Wait for job to reach STITCHING status
	assert.Eventually(t, func() bool {
		req, _ = http.NewRequest("GET", s.baseURL+"/jobs/"+job.ID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, _ = http.DefaultClient.Do(req)
		var jobRes domain.JobResponse
		json.NewDecoder(resp.Body).Decode(&jobRes)
		return jobRes.Job.Status == domain.StatusStitching
	}, 5*time.Second, 500*time.Millisecond)

	// 1.7 Now fetch real stitching task ID from DB
	var stitchTaskID uuid.UUID
	err = s.app.GetDB().QueryRow(ctx, "SELECT id FROM tasks WHERE job_id = $1 AND type = $2 LIMIT 1", job.ID, domain.TypeStitching).Scan(&stitchTaskID)
	require.NoError(t, err)

	// 2. Simulate Stitching Result
	result := domain.RenderResult{
		ProjectID: job.ID,
		TaskID:    stitchTaskID.String(),
		Type:      domain.TypeStitching,
		Status:    domain.StatusCompleted,
		ResultURL: stringPtr("s3://bucket/final.mp4"),
	}
	resBody, _ = json.Marshal(result)
	s.kafkaWriter.WriteMessages(ctx, kafka.Message{Key: []byte(job.ID), Value: resBody})

	assert.Eventually(t, func() bool {
		req, _ = http.NewRequest("GET", s.baseURL+"/jobs/"+job.ID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, _ = http.DefaultClient.Do(req)
		var jobRes domain.JobResponse
		json.NewDecoder(resp.Body).Decode(&jobRes)
		
		return jobRes.Job.ResultURL != nil && strings.Contains(*jobRes.Job.ResultURL, "final.mp4")
	}, 10*time.Second, 500*time.Millisecond)
}

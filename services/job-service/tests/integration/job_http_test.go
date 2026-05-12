//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"

	"kseilons/job-service/internall/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *JobIntegrationTestSuite) Test_CreateAndGetJob() {
	t := s.T()
	token := s.createToken("test-user-1")

	// 1. Create Job
	reqBody := domain.CreateJobRequest{
		Name:        "Integration Test Job",
		MainFileURL: "test/scene.blend",
		WorkerCount: 3,
		FrameStart:  1,
		FrameEnd:    100,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", s.baseURL+"/jobs", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	var job domain.Job
	err = json.NewDecoder(resp.Body).Decode(&job)
	require.NoError(t, err)
	assert.NotEmpty(t, job.ID)

	// 2. Get Job
	req, _ = http.NewRequest("GET", s.baseURL+"/jobs/"+job.ID, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var jobRes domain.JobResponse
	err = json.NewDecoder(resp.Body).Decode(&jobRes)
	require.NoError(t, err)
	assert.Equal(t, job.ID, jobRes.Job.ID)
}

func (s *JobIntegrationTestSuite) Test_ListJobs() {
	t := s.T()
	token := s.createToken("test-user-list")

	req, _ := http.NewRequest("GET", s.baseURL+"/jobs", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var jobs []domain.Job
	json.NewDecoder(resp.Body).Decode(&jobs)
	assert.Empty(t, jobs)
}

func (s *JobIntegrationTestSuite) Test_GenerateUploadURL() {
	t := s.T()
	token := s.createToken("test-user-upload")

	req, _ := http.NewRequest("GET", s.baseURL+"/jobs/upload-url?filename=scene.blend", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var res map[string]string
	json.NewDecoder(resp.Body).Decode(&res)
	assert.Contains(t, res["upload_url"], "http://")
}

func (s *JobIntegrationTestSuite) Test_RetryJob() {
	t := s.T()
	token := s.createToken("test-user-retry")

	jobBody, _ := json.Marshal(domain.CreateJobRequest{
		Name: "Retry Job", MainFileURL: "f", WorkerCount: 1,
	})
	postReq, _ := http.NewRequest("POST", s.baseURL+"/jobs", bytes.NewBuffer(jobBody))
	postReq.Header.Set("Authorization", "Bearer "+token)
	postReq.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(postReq)
	var job domain.Job
	json.NewDecoder(resp.Body).Decode(&job)

	req, _ := http.NewRequest("POST", s.baseURL+"/jobs/"+job.ID+"/retry", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
}

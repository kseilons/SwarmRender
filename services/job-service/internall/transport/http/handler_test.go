package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"kseilons/job-service/internall/domain"
	transport_http "kseilons/job-service/internall/transport/http"
	"kseilons/job-service/internall/transport/http/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandler_CreateJob(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := mocks.NewJobUseCase(t)
	handler := transport_http.NewHandler(mockUC)

	router := gin.New()
	handler.Register(router)

	reqBody := domain.CreateJobRequest{
		Name:        "api-test-job",
		S3Path:      "bucket/project.blend",
		WorkerCount: 3,
	}
	body, _ := json.Marshal(reqBody)

	t.Run("Success", func(t *testing.T) {
		mockUC.On("CreateJob", mock.Anything, reqBody, mock.Anything).
			Return(domain.Job{ID: "job-123", Name: "api-test-job"}, nil)

		req, _ := http.NewRequest(http.MethodPost, "/jobs", bytes.NewBuffer(body))
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusAccepted, resp.Code)
		var job domain.Job
		err := json.Unmarshal(resp.Body.Bytes(), &job)
		assert.NoError(t, err)
		assert.Equal(t, "job-123", job.ID)
	})

	t.Run("BadRequest", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/jobs", bytes.NewBuffer([]byte("{invalid json}")))
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})
}

func TestHandler_GetJob(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := mocks.NewJobUseCase(t)
	handler := transport_http.NewHandler(mockUC)

	router := gin.New()
	handler.Register(router)

	t.Run("Success", func(t *testing.T) {
		mockUC.On("GetJob", mock.Anything, "job-123").
			Return(domain.JobResponse{Job: domain.Job{ID: "job-123"}}, nil)

		req, _ := http.NewRequest(http.MethodGet, "/jobs/job-123", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		var job domain.JobResponse
		_ = json.Unmarshal(resp.Body.Bytes(), &job)
		assert.Equal(t, "job-123", job.ID)
	})

	t.Run("NotFound", func(t *testing.T) {
		mockUC.On("GetJob", mock.Anything, "missing").
			Return(domain.JobResponse{}, assert.AnError)

		req, _ := http.NewRequest(http.MethodGet, "/jobs/missing", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

func TestHandler_ListJobs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := mocks.NewJobUseCase(t)
	handler := transport_http.NewHandler(mockUC)

	router := gin.New()
	handler.Register(router)

	t.Run("Success", func(t *testing.T) {
		mockUC.On("ListJobs", mock.Anything).
			Return([]domain.Job{{ID: "1"}, {ID: "2"}}, nil).Once()

		req, _ := http.NewRequest(http.MethodGet, "/jobs", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		var jobs []domain.Job
		_ = json.Unmarshal(resp.Body.Bytes(), &jobs)
		assert.Len(t, jobs, 2)
	})

	t.Run("Error", func(t *testing.T) {
		mockUC.On("ListJobs", mock.Anything).
			Return(nil, assert.AnError).Once()

		req, _ := http.NewRequest(http.MethodGet, "/jobs", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
	})
}

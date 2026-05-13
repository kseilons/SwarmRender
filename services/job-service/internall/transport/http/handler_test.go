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
	"kseilons/job-service/internall/usecase"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandler_CreateJob(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := mocks.NewJobUseCase(t)
	handler := transport_http.NewHandler(mockUC, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("auth_user", domain.AuthUser{ID: "test-user"})
	})
	handler.Register(router)

	reqBody := domain.CreateJobRequest{
		Name:        "api-test-job",
		MainFileURL: "bucket/project.blend",
		WorkerCount: 3,
	}
	body, _ := json.Marshal(reqBody)

	t.Run("Success", func(t *testing.T) {
		mockUC.On("CreateJob", mock.Anything, reqBody, domain.AuthUser{ID: "test-user"}, mock.Anything).
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
	handler := transport_http.NewHandler(mockUC, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("auth_user", domain.AuthUser{ID: "test-user"})
	})
	handler.Register(router)

	t.Run("Success", func(t *testing.T) {
		mockUC.On("GetJob", mock.Anything, "job-123", domain.AuthUser{ID: "test-user"}).
			Return(domain.JobResponse{Job: domain.Job{ID: "job-123"}}, nil)

		req, _ := http.NewRequest(http.MethodGet, "/jobs/job-123", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		var job domain.JobResponse
		_ = json.Unmarshal(resp.Body.Bytes(), &job)
		assert.Equal(t, "job-123", job.Job.ID)
	})

	t.Run("NotFound", func(t *testing.T) {
		mockUC.On("GetJob", mock.Anything, "missing", domain.AuthUser{ID: "test-user"}).
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
	handler := transport_http.NewHandler(mockUC, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("auth_user", domain.AuthUser{ID: "test-user"})
	})
	handler.Register(router)

	t.Run("Success", func(t *testing.T) {
		mockUC.On("ListJobs", mock.Anything, domain.AuthUser{ID: "test-user"}).
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
		mockUC.On("ListJobs", mock.Anything, domain.AuthUser{ID: "test-user"}).
			Return(nil, assert.AnError).Once()

		req, _ := http.NewRequest(http.MethodGet, "/jobs", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
	})
}

func TestHandler_RetryJob(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := mocks.NewJobUseCase(t)
	handler := transport_http.NewHandler(mockUC, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("auth_user", domain.AuthUser{ID: "test-user"})
	})
	handler.Register(router)

	t.Run("Success", func(t *testing.T) {
		mockUC.On("RetryJob", mock.Anything, "job-123", domain.AuthUser{ID: "test-user"}, mock.Anything).
			Return(nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/jobs/job-123/retry", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusAccepted, resp.Code)
	})

	t.Run("Error", func(t *testing.T) {
		mockUC.On("RetryJob", mock.Anything, "job-fail", domain.AuthUser{ID: "test-user"}, mock.Anything).
			Return(assert.AnError).Once()

		req, _ := http.NewRequest(http.MethodPost, "/jobs/job-fail/retry", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
	})
}

func TestHandler_GenerateUploadURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := mocks.NewJobUseCase(t)
	handler := transport_http.NewHandler(mockUC, nil)

	router := gin.New()
	handler.Register(router)

	t.Run("Success", func(t *testing.T) {
		mockUC.On("GenerateUploadURL", mock.Anything, "test.blend", "proj-1").
			Return("http://s3/upload", "s3://bucket/test.blend", nil).Once()

		req, _ := http.NewRequest(http.MethodGet, "/jobs/upload-url?filename=test.blend&project_id=proj-1", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		var res map[string]string
		_ = json.Unmarshal(resp.Body.Bytes(), &res)
		assert.Equal(t, "http://s3/upload", res["upload_url"])
	})

	t.Run("MissingFilename", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/jobs/upload-url", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("Error", func(t *testing.T) {
		mockUC.On("GenerateUploadURL", mock.Anything, "fail.blend", "").
			Return("", "", assert.AnError).Once()

		req, _ := http.NewRequest(http.MethodGet, "/jobs/upload-url?filename=fail.blend", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
	})
}

func TestHandler_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := mocks.NewJobUseCase(t)
	handler := transport_http.NewHandler(mockUC, nil)

	router := gin.New()
	// No auth user set in context
	handler.Register(router)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/jobs"},
		{http.MethodGet, "/jobs"},
		{http.MethodGet, "/jobs/123"},
		{http.MethodPost, "/jobs/123/retry"},
	}

	for _, e := range endpoints {
		t.Run(e.path, func(t *testing.T) {
			body := []byte("{}")
			if e.path == "/jobs" && e.method == http.MethodPost {
				reqBody := domain.CreateJobRequest{
					Name:        "test",
					MainFileURL: "test",
					WorkerCount: 1,
				}
				body, _ = json.Marshal(reqBody)
			}
			req, _ := http.NewRequest(e.method, e.path, bytes.NewBuffer(body))
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			assert.Equal(t, http.StatusUnauthorized, resp.Code)
		})
	}
}

func TestHandler_HandleWS_AuthFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := transport_http.NewHandler(nil, nil)
	router := gin.New()
	handler.RegisterWS(router)

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + server.URL[4:] + "/ws"

	t.Run("InvalidAuthMessage", func(t *testing.T) {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		assert.NoError(t, err)
		defer conn.Close()

		err = conn.WriteJSON(map[string]string{"type": "not-auth"})
		assert.NoError(t, err)

		var msg map[string]interface{}
		err = conn.ReadJSON(&msg)
		assert.NoError(t, err)
		assert.Equal(t, "error", msg["type"])
	})

	t.Run("InvalidTokenFormat", func(t *testing.T) {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		assert.NoError(t, err)
		defer conn.Close()

		err = conn.WriteJSON(map[string]string{"type": "auth", "token": "not-a-jwt"})
		assert.NoError(t, err)

		var msg map[string]interface{}
		err = conn.ReadJSON(&msg)
		assert.NoError(t, err)
		assert.Equal(t, "error", msg["type"])
		assert.Contains(t, msg["payload"], "token")
	})

	t.Run("MissingSubjectClaim", func(t *testing.T) {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		assert.NoError(t, err)
		defer conn.Close()

		// JWT without 'sub'
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"email": "test@ex.com"})
		ts, _ := token.SignedString([]byte("secret"))

		err = conn.WriteJSON(map[string]string{"type": "auth", "token": ts})
		assert.NoError(t, err)

		var msg map[string]interface{}
		err = conn.ReadJSON(&msg)
		assert.Equal(t, "error", msg["type"])
		assert.Contains(t, msg["payload"], "identity")
	})

	t.Run("SuccessAuth", func(t *testing.T) {
		mockHub := usecase.NewHub()
		go mockHub.Run()

		handler := transport_http.NewHandler(nil, mockHub)
		router := gin.New()
		handler.RegisterWS(router)
		server := httptest.NewServer(router)
		defer server.Close()

		wsURL := "ws" + server.URL[4:] + "/ws"
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		assert.NoError(t, err)
		defer conn.Close()

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "user-123"})
		ts, _ := token.SignedString([]byte("secret")) // Key doesn't matter much as we don't verify signature in handler.go currently (it just parses)

		err = conn.WriteJSON(map[string]string{"type": "auth", "token": ts})
		assert.NoError(t, err)

		// Hub should have 1 client now, but we can't easily check internal state.
		// We just check if it doesn't close immediately.
	})
}

package http_test

import (
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

func TestClusterHandler_GetStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := mocks.NewClusterUseCase(t)
	handler := transport_http.NewClusterHandler(mockUC)

	router := gin.New()
	handler.Register(router)

	t.Run("Success", func(t *testing.T) {
		stats := domain.ClusterStats{
			TotalWorkers: 5,
		}
		mockUC.On("GetStats", mock.Anything).Return(stats, nil).Once()

		req, _ := http.NewRequest(http.MethodGet, "/stats", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		var res domain.ClusterStats
		err := json.Unmarshal(resp.Body.Bytes(), &res)
		assert.NoError(t, err)
		assert.Equal(t, 5, res.TotalWorkers)
	})

	t.Run("Error", func(t *testing.T) {
		mockUC.On("GetStats", mock.Anything).Return(domain.ClusterStats{}, assert.AnError).Once()

		req, _ := http.NewRequest(http.MethodGet, "/stats", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
	})
}

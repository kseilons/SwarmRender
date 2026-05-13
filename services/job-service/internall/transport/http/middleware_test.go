package http_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	transport_http "kseilons/job-service/internall/transport/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Success", func(t *testing.T) {
		r := gin.New()
		r.Use(transport_http.AuthMiddleware())
		r.GET("/test", func(c *gin.Context) {
			user, err := transport_http.GetUser(c)
			assert.NoError(t, err)
			assert.Equal(t, "user-123", user.ID)
			c.Status(http.StatusOK)
		})

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":   "user-123",
			"email": "test@ex.com",
			"realm_access": map[string]interface{}{
				"roles": []interface{}{"user", "admin"},
			},
		})
		ts, _ := token.SignedString([]byte("secret"))

		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts))
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("NoHeader", func(t *testing.T) {
		r := gin.New()
		r.Use(transport_http.AuthMiddleware())
		r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusUnauthorized, resp.Code)
	})

	t.Run("InvalidFormat", func(t *testing.T) {
		r := gin.New()
		r.Use(transport_http.AuthMiddleware())
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "InvalidToken")
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusUnauthorized, resp.Code)
	})

	t.Run("InvalidToken", func(t *testing.T) {
		r := gin.New()
		r.Use(transport_http.AuthMiddleware())
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer not-a-token")
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusUnauthorized, resp.Code)
	})
}

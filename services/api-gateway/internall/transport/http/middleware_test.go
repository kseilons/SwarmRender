package http_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	transport_http "kseilons/api-gateway/internall/transport/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware_Authenticate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 1. Generate RSA Key Pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// 2. Setup JWKS Mock Server
	key, err := jwk.FromRaw(privateKey.PublicKey)
	require.NoError(t, err)
	_ = key.Set(jwk.KeyIDKey, "test-kid")
	_ = key.Set(jwk.AlgorithmKey, "RS256")
	_ = key.Set(jwk.KeyUsageKey, "sig")

	keySet := jwk.NewSet()
	_ = keySet.AddKey(key)

	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(keySet)
	}))
	defer jwksServer.Close()

	// Use discard logger to avoid cluttering test output
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	middleware := transport_http.NewAuthMiddleware(logger, jwksServer.URL)

	// 3. Define Test Cases
	tests := []struct {
		name           string
		setupHeader    func(r *http.Request)
		expectedStatus int
		expectUserID   string
	}{
		{
			name: "Success with valid token",
			setupHeader: func(r *http.Request) {
				token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
					"sub": "user-123",
					"exp": time.Now().Add(time.Hour).Unix(),
				})
				token.Header["kid"] = "test-kid"
				signed, _ := token.SignedString(privateKey)
				r.Header.Set("Authorization", "Bearer "+signed)
			},
			expectedStatus: http.StatusOK,
			expectUserID:   "user-123",
		},
		{
			name:           "Missing Authorization header",
			setupHeader:    func(r *http.Request) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Invalid token format",
			setupHeader: func(r *http.Request) {
				r.Header.Set("Authorization", "InvalidFormat")
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Expired token",
			setupHeader: func(r *http.Request) {
				token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
					"sub": "user-123",
					"exp": time.Now().Add(-time.Hour).Unix(),
				})
				token.Header["kid"] = "test-kid"
				signed, _ := token.SignedString(privateKey)
				r.Header.Set("Authorization", "Bearer "+signed)
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(middleware.Authenticate())
			router.GET("/test", func(c *gin.Context) {
				userID, _ := c.Get("user_id")
				c.JSON(http.StatusOK, gin.H{"user_id": userID})
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			tt.setupHeader(req)

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectUserID != "" {
				var body map[string]interface{}
				_ = json.Unmarshal(w.Body.Bytes(), &body)
				assert.Equal(t, tt.expectUserID, body["user_id"])
			}
		})
	}
}

package http_test

import (
	"context"
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

func setupAuthTest(t *testing.T) (*rsa.PrivateKey, jwk.Set, *httptest.Server) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

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

	return privateKey, keySet, jwksServer
}

func TestAuthMiddleware_Authenticate_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	privateKey, _, jwksServer := setupAuthTest(t)
	defer jwksServer.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	middleware := transport_http.NewAuthMiddleware(logger, jwksServer.URL)

	router := gin.New()
	router.Use(middleware.Authenticate())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub": "user-123",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	token.Header["kid"] = "test-kid"
	signed, _ := token.SignedString(privateKey)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+signed)

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_Authenticate_WebSocketFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	privateKey, _, jwksServer := setupAuthTest(t)
	defer jwksServer.Close()

	middleware := transport_http.NewAuthMiddleware(slog.New(slog.NewTextHandler(io.Discard, nil)), jwksServer.URL)
	router := gin.New()
	router.Use(middleware.Authenticate())
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"sub": "user-123"})
	token.Header["kid"] = "test-kid"
	signed, _ := token.SignedString(privateKey)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Sec-WebSocket-Protocol", "access_token, "+signed)

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_Authenticate_MissingAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	middleware := transport_http.NewAuthMiddleware(slog.New(slog.NewTextHandler(io.Discard, nil)), "")
	router := gin.New()
	router.Use(middleware.Authenticate())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_Authenticate_JWKSFetchError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer jwksServer.Close()

	middleware := transport_http.NewAuthMiddleware(slog.New(slog.NewTextHandler(io.Discard, nil)), jwksServer.URL)
	router := gin.New()
	router.Use(middleware.Authenticate())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer some-token")

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAuthMiddleware_Authenticate_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, _, jwksServer := setupAuthTest(t)
	defer jwksServer.Close()

	middleware := transport_http.NewAuthMiddleware(slog.New(slog.NewTextHandler(io.Discard, nil)), jwksServer.URL)
	router := gin.New()
	router.Use(middleware.Authenticate())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_Authenticate_WrongAlgorithm(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, _, jwksServer := setupAuthTest(t)
	defer jwksServer.Close()

	middleware := transport_http.NewAuthMiddleware(slog.New(slog.NewTextHandler(io.Discard, nil)), jwksServer.URL)
	router := gin.New()
	router.Use(middleware.Authenticate())

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "user-123"})
	signed, _ := token.SignedString([]byte("secret"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+signed)

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_GetJWKS(t *testing.T) {
	_, _, jwksServer := setupAuthTest(t)
	defer jwksServer.Close()

	middleware := transport_http.NewAuthMiddleware(slog.Default(), jwksServer.URL)
	keySet, err := middleware.GetJWKS(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, keySet)
	
	key, found := keySet.LookupKeyID("test-kid")
	assert.True(t, found)
	assert.NotNil(t, key)
}

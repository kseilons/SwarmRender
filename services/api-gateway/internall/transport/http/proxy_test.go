package http_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	transport_http "kseilons/api-gateway/internall/transport/http"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxyHandler_ProxyAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 1. Setup Backend Mock Server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/login", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer backend.Close()

	// 2. Setup Proxy Handler
	logger := slog.Default()
	handler, err := transport_http.NewProxyHandler(logger, backend.URL, "http://unused:8081")
	require.NoError(t, err)

	router := gin.New()
	router.Any("/auth/*path", handler.ProxyAuth())

	proxyServer := httptest.NewServer(router)
	defer proxyServer.Close()

	// 3. Test Request
	resp, err := http.Post(proxyServer.URL+"/auth/login", "application/json", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "ok", string(body))
}

func TestProxyHandler_ProxyJob(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 1. Setup Backend Mock Server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/jobs/list", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer backend.Close()

	// 2. Setup Proxy Handler
	logger := slog.Default()
	handler, err := transport_http.NewProxyHandler(logger, "http://unused:8082", backend.URL)
	require.NoError(t, err)

	router := gin.New()
	router.Any("/jobs/*path", handler.ProxyJob())

	proxyServer := httptest.NewServer(router)
	defer proxyServer.Close()

	// 3. Test Request
	resp, err := http.Get(proxyServer.URL + "/jobs/list")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "ok", string(body))
}

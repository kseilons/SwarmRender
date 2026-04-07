package http

import (
	"log/slog"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

type ProxyHandler struct {
	logger *slog.Logger
	auth   *url.URL
	job    *url.URL
}

func NewProxyHandler(logger *slog.Logger, authURL string, jobURL string) (*ProxyHandler, error) {
	a, err := url.Parse(authURL)
	if err != nil {
		return nil, err
	}
	j, err := url.Parse(jobURL)
	if err != nil {
		return nil, err
	}

	return &ProxyHandler{
		logger: logger,
		auth:   a,
		job:    j,
	}, nil
}

func (h *ProxyHandler) ProxyAuth() gin.HandlerFunc {
	return h.createProxy(h.auth, "/auth")
}

func (h *ProxyHandler) ProxyJob() gin.HandlerFunc {
	return h.createProxy(h.job, "")
}

func (h *ProxyHandler) createProxy(target *url.URL, prefixToRemove string) gin.HandlerFunc {
	proxy := httputil.NewSingleHostReverseProxy(target)

	return func(c *gin.Context) {
		h.logger.Debug("proxying request", slog.String("path", c.Request.URL.Path), slog.String("target", target.String()))

		remote, err := url.Parse(target.String())
		if err != nil {
			panic(err)
		}

		c.Request.Host = remote.Host
		c.Request.URL.Scheme = remote.Scheme
		c.Request.URL.Host = remote.Host

		if prefixToRemove != "" {
			c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, prefixToRemove)
		}

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

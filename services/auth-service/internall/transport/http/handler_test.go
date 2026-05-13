package http_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"kseilons/auth-service/internall/domain"
	transport_http "kseilons/auth-service/internall/transport/http"
	"kseilons/auth-service/internall/transport/http/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandler_Login_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := mocks.NewAuthUseCase(t)
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := transport_http.NewHandler(mockUC, log)

	router := gin.New()
	handler.Register(router)

	req := domain.LoginRequest{Username: "user", Password: "pass"}
	body, _ := json.Marshal(req)

	mockUC.On("Login", mock.Anything, req.Username, req.Password).
		Return(&domain.TokenResponse{AccessToken: "at", RefreshToken: "rt"}, nil)

	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var res domain.TokenResponse
	err := json.Unmarshal(w.Body.Bytes(), &res)
	assert.NoError(t, err)
	assert.Equal(t, "at", res.AccessToken)
}

func TestHandler_Login_Failure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := mocks.NewAuthUseCase(t)
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := transport_http.NewHandler(mockUC, log)

	router := gin.New()
	handler.Register(router)

	mockUC.On("Login", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("auth failed"))

	req := domain.LoginRequest{Username: "u", Password: "p"}
	body, _ := json.Marshal(req)

	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Register_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := mocks.NewAuthUseCase(t)
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := transport_http.NewHandler(mockUC, log)

	router := gin.New()
	handler.Register(router)

	req := domain.RegisterRequest{Username: "new", Email: "new@ex.com", Password: "p"}
	body, _ := json.Marshal(req)

	mockUC.On("Register", mock.Anything, req).Return(nil)

	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_Refresh_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := mocks.NewAuthUseCase(t)
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := transport_http.NewHandler(mockUC, log)

	router := gin.New()
	handler.Register(router)

	req := struct {
		RefreshToken string `json:"refresh_token"`
	}{RefreshToken: "old"}
	body, _ := json.Marshal(req)

	mockUC.On("RefreshToken", mock.Anything, "old").
		Return(&domain.TokenResponse{AccessToken: "new"}, nil)

	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBuffer(body))
	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Health(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := transport_http.NewHandler(nil, slog.Default())
	router := gin.New()
	handler.Register(router)

	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Register_Failure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := mocks.NewAuthUseCase(t)
	handler := transport_http.NewHandler(mockUC, slog.Default())

	router := gin.New()
	handler.Register(router)

	t.Run("UseCaseError", func(t *testing.T) {
		req := domain.RegisterRequest{Username: "u", Email: "u@u.com", Password: "p"}
		mockUC.On("Register", mock.Anything, req).Return(errors.New("fail")).Once()
		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		r, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		router.ServeHTTP(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("BadRequest", func(t *testing.T) {
		w := httptest.NewRecorder()
		r, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer([]byte("invalid")))
		router.ServeHTTP(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_Refresh_Failure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := mocks.NewAuthUseCase(t)
	handler := transport_http.NewHandler(mockUC, slog.Default())

	router := gin.New()
	handler.Register(router)

	t.Run("UseCaseError", func(t *testing.T) {
		mockUC.On("RefreshToken", mock.Anything, "bad").Return(nil, errors.New("fail")).Once()
		body, _ := json.Marshal(map[string]string{"refresh_token": "bad"})
		w := httptest.NewRecorder()
		r, _ := http.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBuffer(body))
		router.ServeHTTP(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("MissingToken", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{})
		w := httptest.NewRecorder()
		r, _ := http.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBuffer(body))
		router.ServeHTTP(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

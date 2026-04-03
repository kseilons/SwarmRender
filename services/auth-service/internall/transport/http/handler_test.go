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

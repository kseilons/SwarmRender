//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"

	"kseilons/auth-service/internall/domain"

	"github.com/stretchr/testify/assert"
)

func (s *AuthIntegrationTestSuite) Test_AuthLifecycle() {
	t := s.T()
	
	// 1. Register User
	regReq := domain.RegisterRequest{
		Username:  "testuser",
		Password:  "testpass123",
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
	}
	body, _ := json.Marshal(regReq)
	resp, err := http.Post(s.baseURL+"/auth/register", "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// 2. Login
	loginReq := domain.LoginRequest{
		Username: "testuser",
		Password: "testpass123",
	}
	body, _ = json.Marshal(loginReq)
	resp, err = http.Post(s.baseURL+"/auth/login", "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var token domain.TokenResponse
	err = json.NewDecoder(resp.Body).Decode(&token)
	assert.NoError(t, err)
	assert.NotEmpty(t, token.AccessToken)
	assert.NotEmpty(t, token.RefreshToken)

	// 3. Refresh Token
	refreshReq := struct {
		RefreshToken string `json:"refresh_token"`
	}{
		RefreshToken: token.RefreshToken,
	}
	body, _ = json.Marshal(refreshReq)
	resp, err = http.Post(s.baseURL+"/auth/refresh", "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var newToken domain.TokenResponse
	err = json.NewDecoder(resp.Body).Decode(&newToken)
	assert.NoError(t, err)
	assert.NotEmpty(t, newToken.AccessToken)
}

func (s *AuthIntegrationTestSuite) Test_Login_InvalidCredentials() {
	t := s.T()
	
	loginReq := domain.LoginRequest{
		Username: "testuser",
		Password: "wrongpassword",
	}
	body, _ := json.Marshal(loginReq)
	resp, err := http.Post(s.baseURL+"/auth/login", "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

package usecase_test

import (
	"context"
	"errors"
	"testing"

	"kseilons/auth-service/internall/config"
	"kseilons/auth-service/internall/domain"
	"kseilons/auth-service/internall/usecase"
	"kseilons/auth-service/internall/usecase/mocks"

	"github.com/Nerzal/gocloak/v13"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAuthUseCase_Login_Success(t *testing.T) {
	mockClient := mocks.NewKeycloakClient(t)
	cfg := config.KeycloakConfig{
		Realm:        "test-realm",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
	}
	uc := usecase.NewAuthUseCase(mockClient, cfg)

	ctx := context.Background()
	username, password := "user", "pass"

	mockClient.On("Login", ctx, cfg.ClientID, cfg.ClientSecret, cfg.Realm, username, password).
		Return(&gocloak.JWT{AccessToken: "access", RefreshToken: "refresh", ExpiresIn: 3600}, nil)

	res, err := uc.Login(ctx, username, password)

	assert.NoError(t, err)
	assert.Equal(t, "access", res.AccessToken)
	assert.Equal(t, "refresh", res.RefreshToken)
}

func TestAuthUseCase_Login_Failure(t *testing.T) {
	mockClient := mocks.NewKeycloakClient(t)
	cfg := config.KeycloakConfig{Realm: "r", ClientID: "c", ClientSecret: "s"}
	uc := usecase.NewAuthUseCase(mockClient, cfg)

	ctx := context.Background()
	mockClient.On("Login", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("auth error"))

	res, err := uc.Login(ctx, "u", "p")

	assert.Error(t, err)
	assert.Nil(t, res)
}

func TestAuthUseCase_Register_Success(t *testing.T) {
	mockClient := mocks.NewKeycloakClient(t)
	cfg := config.KeycloakConfig{Realm: "test-realm"}
	uc := usecase.NewAuthUseCase(mockClient, cfg)

	ctx := context.Background()
	req := domain.RegisterRequest{
		Username: "newuser",
		Email:    "new@example.com",
		Password: "password",
	}

	mockClient.On("LoginAdmin", ctx, "admin", "admin", "master").
		Return(&gocloak.JWT{AccessToken: "admin-token"}, nil)
	mockClient.On("CreateUser", ctx, "admin-token", cfg.Realm, mock.MatchedBy(func(u gocloak.User) bool {
		return *u.Username == req.Username && *u.Email == req.Email
	})).Return("user-id", nil)
	mockClient.On("SetPassword", ctx, "admin-token", "user-id", cfg.Realm, req.Password, false).
		Return(nil)

	err := uc.Register(ctx, req)

	assert.NoError(t, err)
}

func TestAuthUseCase_RefreshToken_Success(t *testing.T) {
	mockClient := mocks.NewKeycloakClient(t)
	cfg := config.KeycloakConfig{Realm: "r", ClientID: "c", ClientSecret: "s"}
	uc := usecase.NewAuthUseCase(mockClient, cfg)

	ctx := context.Background()
	mockClient.On("RefreshToken", ctx, "old-refresh", cfg.ClientID, cfg.ClientSecret, cfg.Realm).
		Return(&gocloak.JWT{AccessToken: "new-access", RefreshToken: "new-refresh"}, nil)

	res, err := uc.RefreshToken(ctx, "old-refresh")

	assert.NoError(t, err)
	assert.Equal(t, "new-access", res.AccessToken)
}

package usecase

import (
	"context"
	"fmt"
	"kseilons/auth-service/internall/config"
	"kseilons/auth-service/internall/domain"

	"github.com/Nerzal/gocloak/v13"
)

//go:generate mockery --name KeycloakClient --output ./mocks --case underscore
type KeycloakClient interface {
	Login(ctx context.Context, clientID, clientSecret, realm, username, password string) (*gocloak.JWT, error)
	LoginAdmin(ctx context.Context, username, password, realm string) (*gocloak.JWT, error)
	CreateUser(ctx context.Context, token, realm string, user gocloak.User) (string, error)
	SetPassword(ctx context.Context, token, userID, realm, password string, temporary bool) error
	RefreshToken(ctx context.Context, refreshToken, clientID, clientSecret, realm string) (*gocloak.JWT, error)
}

type AuthUseCase struct {
	client KeycloakClient
	cfg    config.KeycloakConfig
}

func NewAuthUseCase(client KeycloakClient, cfg config.KeycloakConfig) *AuthUseCase {
	return &AuthUseCase{
		client: client,
		cfg:    cfg,
	}
}

func (u *AuthUseCase) Login(ctx context.Context, username, password string) (*domain.TokenResponse, error) {
	token, err := u.client.Login(ctx, u.cfg.ClientID, u.cfg.ClientSecret, u.cfg.Realm, username, password)
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	return &domain.TokenResponse{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresIn:    token.ExpiresIn,
	}, nil
}

func (u *AuthUseCase) Register(ctx context.Context, req domain.RegisterRequest) error {
	// 1. Get Admin Token
	token, err := u.client.LoginAdmin(ctx, "admin", "admin", "master")
	if err != nil {
		return fmt.Errorf("failed to login as admin: %w", err)
	}

	// 2. Create User
	user := gocloak.User{
		Username:      gocloak.StringP(req.Username),
		Email:         gocloak.StringP(req.Email),
		FirstName:     gocloak.StringP(req.FirstName),
		LastName:      gocloak.StringP(req.LastName),
		Enabled:       gocloak.BoolP(true),
		EmailVerified: gocloak.BoolP(true),
	}

	userID, err := u.client.CreateUser(ctx, token.AccessToken, u.cfg.Realm, user)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	// 3. Set Password
	err = u.client.SetPassword(ctx, token.AccessToken, userID, u.cfg.Realm, req.Password, false)
	if err != nil {
		return fmt.Errorf("failed to set password: %w", err)
	}

	return nil
}

func (u *AuthUseCase) RefreshToken(ctx context.Context, refreshToken string) (*domain.TokenResponse, error) {
	token, err := u.client.RefreshToken(ctx, refreshToken, u.cfg.ClientID, u.cfg.ClientSecret, u.cfg.Realm)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	return &domain.TokenResponse{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresIn:    token.ExpiresIn,
	}, nil
}

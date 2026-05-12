//go:build integration

package integration

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"kseilons/auth-service/internall/app"
	"kseilons/auth-service/internall/config"
	"kseilons/auth-service/internall/testutil"

	"github.com/Nerzal/gocloak/v13"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type AuthIntegrationTestSuite struct {
	suite.Suite
	env     *testutil.TestEnv
	app     *app.App
	baseURL string
}

func (s *AuthIntegrationTestSuite) SetupSuite() {
	ctx := context.Background()

	// 1. Setup Infrastructure
	env, err := testutil.SetupIntegrationEnv(ctx)
	require.NoError(s.T(), err)
	s.env = env

	keycloakURL, err := env.Keycloak.GetAuthServerURL(ctx)
	require.NoError(s.T(), err)

	// 2. Setup Keycloak Client for App
	client := gocloak.NewClient(keycloakURL)
	token, err := client.LoginAdmin(ctx, "admin", "admin", "master")
	require.NoError(s.T(), err)

	clientID := "auth-service-client"
	clientSecret := "very-secret-secret"

	_, err = client.CreateClient(ctx, token.AccessToken, "master", gocloak.Client{
		ClientID:                  gocloak.StringP(clientID),
		Secret:                    gocloak.StringP(clientSecret),
		Enabled:                   gocloak.BoolP(true),
		PublicClient:              gocloak.BoolP(false),
		ServiceAccountsEnabled:    gocloak.BoolP(true),
		DirectAccessGrantsEnabled: gocloak.BoolP(true),
	})
	require.NoError(s.T(), err)

	// 3. Setup App Config
	cfg := &config.Config{
		Env: "test",
		HTTP: config.HTTPConfig{
			Port: 8083,
		},
		Keycloak: config.KeycloakConfig{
			BaseURL:      keycloakURL,
			Realm:        "master",
			ClientID:     clientID,
			ClientSecret: clientSecret,
		},
	}

	s.baseURL = "http://localhost:8083"

	// 4. Start App
	s.app = app.New(cfg, slog.Default())
	
	appCtx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := s.app.Run(appCtx); err != nil {
			s.T().Logf("app run error: %v", err)
		}
	}()
	s.T().Cleanup(cancel)

	// Wait for app to be ready
	require.Eventually(s.T(), func() bool {
		resp, err := http.Get(s.baseURL + "/health")
		return err == nil && resp.StatusCode == http.StatusOK
	}, 10*time.Second, 500*time.Millisecond)
}

func (s *AuthIntegrationTestSuite) TearDownSuite() {
	ctx := context.Background()
	if s.app != nil {
		s.app.Shutdown()
	}
	if s.env != nil {
		s.env.Terminate(ctx)
	}
}

func TestAuthIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(AuthIntegrationTestSuite))
}

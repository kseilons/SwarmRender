package testutil

import (
	"context"
	"fmt"

	"github.com/stillya/testcontainers-keycloak"
)

type TestEnv struct {
	Keycloak *keycloak.KeycloakContainer
}

func SetupIntegrationEnv(ctx context.Context) (*TestEnv, error) {
	// 1. Keycloak
	keycloakContainer, err := keycloak.Run(ctx,
		"quay.io/keycloak/keycloak:22.0.1",
		keycloak.WithAdminUsername("admin"),
		keycloak.WithAdminPassword("admin"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start keycloak: %w", err)
	}

	return &TestEnv{
		Keycloak: keycloakContainer,
	}, nil
}

func (e *TestEnv) Terminate(ctx context.Context) {
	if e.Keycloak != nil {
		e.Keycloak.Terminate(ctx)
	}
}

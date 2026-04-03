package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad_Success(t *testing.T) {
	content := `
env: "prod"
http:
  port: 8080
keycloak:
  base_url: "http://keycloak:8080"
  realm: "my-realm"
  client_id: "my-client"
  client_secret: "my-secret"
`
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)
	tmpFile.Close()

	cfg, err := Load(tmpFile.Name())
	assert.NoError(t, err)

	assert.Equal(t, "prod", cfg.Env)
	assert.Equal(t, 8080, cfg.HTTP.Port)
	assert.Equal(t, "http://keycloak:8080", cfg.Keycloak.BaseURL)
	assert.Equal(t, "my-realm", cfg.Keycloak.Realm)
}

func TestLoad_EnvOverride(t *testing.T) {
	content := `
keycloak:
  client_id: "id"
  client_secret: "secret"
`
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)
	tmpFile.Close()

	os.Setenv("ENV", "dev")
	os.Setenv("HTTP_PORT", "9090")
	defer os.Unsetenv("ENV")
	defer os.Unsetenv("HTTP_PORT")

	cfg, err := Load(tmpFile.Name())
	assert.NoError(t, err)

	assert.Equal(t, "dev", cfg.Env)
	assert.Equal(t, 9090, cfg.HTTP.Port)
}

func TestLoad_MissingRequired(t *testing.T) {
	content := `env: "local"`
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)
	tmpFile.Close()

	cfg, err := Load(tmpFile.Name())
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

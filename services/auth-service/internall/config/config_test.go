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

func TestLoad_NoFile(t *testing.T) {
	cfg, err := Load("non-existent-path.yaml")
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoad_InvalidYaml(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config-invalid-*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, _ = tmpFile.WriteString("invalid: [yaml: structure")
	tmpFile.Close()

	_, err = Load(tmpFile.Name())
	assert.Error(t, err)
}

func TestLoad_DefaultPath(t *testing.T) {
	content := "keycloak:\n  client_id: \"c\"\n  client_secret: \"s\""

	// Create a temporary file to act as the default config
	tmpFile, _ := os.CreateTemp("", "config-*.yaml")
	defer os.Remove(tmpFile.Name())
	_, _ = tmpFile.WriteString(content)
	tmpFile.Close()

	os.Setenv("CONFIG_PATH", tmpFile.Name())
	defer os.Unsetenv("CONFIG_PATH")

	cfg, err := Load("")
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
}

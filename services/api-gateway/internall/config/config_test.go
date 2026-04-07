package config_test

import (
	"os"
	"testing"

	"kseilons/api-gateway/internall/config"

	"github.com/stretchr/testify/assert"
)

func TestLoad_Success(t *testing.T) {
	content := `
env: "prod"
http:
  port: 9000
services:
  auth_service: "http://auth:8082"
  job_service: "http://job:8081"
keycloak:
  jwks_url: "http://keycloak:8080/certs"
`
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)
	tmpFile.Close()

	cfg, err := config.Load(tmpFile.Name())
	assert.NoError(t, err)

	assert.Equal(t, "prod", cfg.Env)
	assert.Equal(t, 9000, cfg.HTTP.Port)
	assert.Equal(t, "http://auth:8082", cfg.Services.AuthService)
}

func TestLoad_EnvOverride(t *testing.T) {
	content := `
env: "local"
http:
  port: 8000
`
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)
	tmpFile.Close()

	os.Setenv("ENV", "dev")
	os.Setenv("HTTP_PORT", "7777")
	defer os.Unsetenv("ENV")
	defer os.Unsetenv("HTTP_PORT")

	cfg, err := config.Load(tmpFile.Name())
	assert.NoError(t, err)

	assert.Equal(t, "dev", cfg.Env)
	assert.Equal(t, 7777, cfg.HTTP.Port)
}

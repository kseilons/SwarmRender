package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"kseilons/api-gateway/internall/config"

	"github.com/stretchr/testify/assert"
)

func TestLoad_Success(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	content := `
env: "production"
http:
  port: 8080
services:
  auth_service: "http://auth:8082"
  job_service: "http://job:8081"
`
	err := os.WriteFile(configPath, []byte(content), 0644)
	assert.NoError(t, err)

	cfg, err := config.Load(configPath)
	assert.NoError(t, err)
	assert.Equal(t, "production", cfg.Env)
	assert.Equal(t, 8080, cfg.HTTP.Port)
}

func TestLoad_FromConfigPathEnv(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config_env.yaml")
	content := `env: "staging"`
	os.WriteFile(configPath, []byte(content), 0644)

	os.Setenv("CONFIG_PATH", configPath)
	defer os.Unsetenv("CONFIG_PATH")

	cfg, err := config.Load("")
	assert.NoError(t, err)
	assert.Equal(t, "staging", cfg.Env)
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := config.Load("non-existent.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config file does not exist")
}

func TestLoad_ReadConfigError(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")
	os.WriteFile(configPath, []byte("invalid: yaml: :"), 0644)

	_, err := config.Load(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot read config")
}

func TestMustLoad_Success(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "must_load.yaml")
	os.WriteFile(configPath, []byte("env: test"), 0644)

	os.Setenv("CONFIG_PATH", configPath)
	defer os.Unsetenv("CONFIG_PATH")

	assert.NotPanics(t, func() {
		cfg := config.MustLoad()
		assert.Equal(t, "test", cfg.Env)
	})
}

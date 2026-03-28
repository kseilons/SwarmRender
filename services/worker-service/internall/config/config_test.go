package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad_Success(t *testing.T) {
	// Create a temporary config file
	content := `
env: "prod"
blender:
  path: "/usr/bin/blender"
  allowed_devices: ["CUDA"]
kafka:
  brokers: ["localhost:9092"]
minio:
  endpoint: "minio:9000"
  access_key: "admin"
  secret_key: "password"
`
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)
	tmpFile.Close()

	// Load config
	cfg, err := Load(tmpFile.Name())
	assert.NoError(t, err)

	assert.Equal(t, "prod", cfg.Env)
	assert.Equal(t, "/usr/bin/blender", cfg.Blender.Path)
	assert.Equal(t, []string{"CUDA"}, cfg.Blender.AllowedDevices)
	assert.Equal(t, "minio:9000", cfg.Minio.Endpoint)
}

func TestLoad_EnvOverride(t *testing.T) {
	content := `
blender:
  path: "/usr/bin/blender"
minio:
  access_key: "admin"
  secret_key: "password"
`
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)
	tmpFile.Close()

	// Set env override
	os.Setenv("ENV", "staging")
	defer os.Unsetenv("ENV")

	cfg, err := Load(tmpFile.Name())
	assert.NoError(t, err)

	assert.Equal(t, "staging", cfg.Env)
}

func TestLoad_FileNotFound(t *testing.T) {
	cfg, err := Load("non-existent.yaml")
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoad_MissingRequired(t *testing.T) {
	// Missing blender.path and minio keys
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

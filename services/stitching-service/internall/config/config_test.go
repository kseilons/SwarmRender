package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"kseilons/stitching-service/internall/config"

	"github.com/stretchr/testify/assert"
)

func TestLoad_FromEnv(t *testing.T) {
	os.Setenv("ENV", "test")
	os.Setenv("S3_ACCESS_KEY", "access")
	os.Setenv("S3_SECRET_KEY", "secret")
	defer os.Unsetenv("ENV")
	defer os.Unsetenv("S3_ACCESS_KEY")
	defer os.Unsetenv("S3_SECRET_KEY")

	cfg, err := config.Load("non-existent.yaml")
	assert.NoError(t, err)
	assert.Equal(t, "test", cfg.Env)
	assert.Equal(t, "access", cfg.Minio.AccessKey)
}

func TestLoad_FromConfigPathEnv(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	content := `
env: "production"
minio:
  access_key: "file_access"
  secret_key: "file_secret"
`
	err := os.WriteFile(configPath, []byte(content), 0644)
	assert.NoError(t, err)

	os.Setenv("CONFIG_PATH", configPath)
	defer os.Unsetenv("CONFIG_PATH")

	cfg, err := config.Load("")
	assert.NoError(t, err)
	assert.Equal(t, "production", cfg.Env)
	assert.Equal(t, "file_access", cfg.Minio.AccessKey)
}

func TestLoad_FromFileSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "direct_config.yaml")
	content := `
env: "dev"
minio:
  access_key: "direct_access"
  secret_key: "direct_secret"
`
	os.WriteFile(configPath, []byte(content), 0644)

	cfg, err := config.Load(configPath)
	assert.NoError(t, err)
	assert.Equal(t, "dev", cfg.Env)
}

func TestLoad_FileReadError(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid_config.yaml")
	content := "env: production\n  minio: invalid"
	os.WriteFile(configPath, []byte(content), 0644)

	_, err := config.Load(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot read config")
}

func TestLoad_ErrorMissingRequiredEnv(t *testing.T) {
	os.Unsetenv("S3_ACCESS_KEY")
	os.Unsetenv("S3_SECRET_KEY")

	_, err := config.Load("non-existent.yaml")
	assert.Error(t, err)
}

func TestMustLoad_Success(t *testing.T) {
	os.Setenv("S3_ACCESS_KEY", "access")
	os.Setenv("S3_SECRET_KEY", "secret")
	os.Setenv("CONFIG_PATH", "non-existent-but-env-works.yaml")
	defer os.Unsetenv("S3_ACCESS_KEY")
	defer os.Unsetenv("S3_SECRET_KEY")
	defer os.Unsetenv("CONFIG_PATH")

	assert.NotPanics(t, func() {
		cfg := config.MustLoad()
		assert.NotNil(t, cfg)
	})
}

package config

import (
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env     string        `yaml:"env" env:"ENV" env-default:"local"`
	Blender BlenderConfig `yaml:"blender"`
}

type BlenderConfig struct {
	Path           string   `yaml:"path" env:"BLENDER_PATH" env-required:"true"`
	AllowedDevices []string `yaml:"allowed_devices" env:"ALLOWED_DEVICES" env-default:"CPU"`
	Threads        int      `yaml:"threads" env:"RENDER_THREADS" env-default:"0"`
}

// MustLoad loads the configuration. It panics if something goes wrong.
// Usage: cfg := config.MustLoad()
func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file does not exist: %s", configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config: %s", err)
	}

	return &cfg
}

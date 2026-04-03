package config

import (
	"fmt"
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env      string         `yaml:"env" env:"ENV" env-default:"local"`
	HTTP     HTTPConfig     `yaml:"http"`
	Keycloak KeycloakConfig `yaml:"keycloak"`
}

type HTTPConfig struct {
	Port int `yaml:"port" env:"HTTP_PORT" env-default:"8082"`
}

type KeycloakConfig struct {
	BaseURL      string `yaml:"base_url" env:"KEYCLOAK_BASE_URL" env-default:"http://localhost:8080"`
	Realm        string `yaml:"realm" env:"KEYCLOAK_REALM" env-default:"master"`
	ClientID     string `yaml:"client_id" env:"KEYCLOAK_CLIENT_ID" env-required:"true"`
	ClientSecret string `yaml:"client_secret" env:"KEYCLOAK_CLIENT_SECRET" env-required:"true"`
}

// Load loads the configuration from a file and environment variables.
func Load(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = os.Getenv("CONFIG_PATH")
		if configPath == "" {
			configPath = "config/config.yaml"
		}
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist: %s", configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		return nil, fmt.Errorf("cannot read config: %w", err)
	}

	return &cfg, nil
}

// MustLoad loads the configuration. It panics if something goes wrong.
func MustLoad() *Config {
	cfg, err := Load("")
	if err != nil {
		log.Fatal(err)
	}

	return cfg
}

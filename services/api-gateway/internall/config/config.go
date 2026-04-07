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
	Services ServicesConfig `yaml:"services"`
	Keycloak KeycloakConfig `yaml:"keycloak"`
}

type HTTPConfig struct {
	Port int `yaml:"port" env:"HTTP_PORT" env-default:"8000"`
}

type ServicesConfig struct {
	AuthService string `yaml:"auth_service" env:"AUTH_SERVICE_URL" env-default:"http://auth-service:8082"`
	JobService  string `yaml:"job_service" env:"JOB_SERVICE_URL" env-default:"http://job-service:8081"`
}

type KeycloakConfig struct {
	JWKSURL string `yaml:"jwks_url" env:"KEYCLOAK_JWKS_URL" env-default:"http://keycloak:8080/realms/render-farm/protocol/openid-connect/certs"`
}

func Load(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = os.Getenv("CONFIG_PATH")
		if configPath == "" {
			configPath = "config/config.yaml"
		}
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist: %s", configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		return nil, fmt.Errorf("cannot read config: %w", err)
	}

	return &cfg, nil
}

func MustLoad() *Config {
	cfg, err := Load("")
	if err != nil {
		log.Fatal(err)
	}

	return cfg
}

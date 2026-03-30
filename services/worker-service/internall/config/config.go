package config

import (
	"fmt"
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env     string        `yaml:"env" env:"ENV" env-default:"local"`
	Blender BlenderConfig `yaml:"blender"`
	Kafka   KafkaConfig   `yaml:"kafka"`
	Minio   MinioConfig   `yaml:"minio"`
}

type BlenderConfig struct {
	Path           string   `yaml:"path" env:"BLENDER_PATH" env-required:"true"`
	AllowedDevices []string `yaml:"allowed_devices" env:"ALLOWED_DEVICES" env-default:"CPU"`
	Threads        int      `yaml:"threads" env:"RENDER_THREADS" env-default:"0"`
}

type KafkaConfig struct {
	Brokers       []string `yaml:"brokers" env:"KAFKA_BROKERS" env-default:"localhost:9092"`
	RenderTopic   string   `yaml:"render_topic" env:"KAFKA_RENDER_TOPIC" env-default:"render-tasks"`
	MetadataTopic string   `yaml:"metadata_topic" env:"KAFKA_METADATA_TOPIC" env-default:"metadata-tasks"`
	ResultTopic   string   `yaml:"result_topic" env:"KAFKA_RESULT_TOPIC" env-default:"render-results"`
	GroupID       string   `yaml:"group_id" env:"KAFKA_GROUP_ID" env-default:"worker-service"`
}

type MinioConfig struct {
	Endpoint  string `yaml:"endpoint" env:"MINIO_ENDPOINT" env-default:"localhost:9000"`
	AccessKey string `yaml:"access_key" env:"MINIO_ACCESS_KEY" env-required:"true"`
	SecretKey string `yaml:"secret_key" env:"MINIO_SECRET_KEY" env-required:"true"`
	UseSSL    bool   `yaml:"use_ssl" env:"MINIO_USE_SSL" env-default:"false"`
	Bucket    string `yaml:"bucket" env:"MINIO_BUCKET" env-default:"blender-renders"`
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

package config

import (
	"fmt"
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env   string           `yaml:"env" env:"ENV" env-default:"local"`
	Http  HttpServerConfig `yaml:"http"`
	Kafka KafkaConfig      `yaml:"kafka"`
	Minio MinioConfig      `yaml:"minio"`
}

type HttpServerConfig struct {
	Address string `yaml:"address" env:"HTTP_ADDRESS" env-default:"0.0.0.0:8082"`
}

type KafkaConfig struct {
	Brokers       []string `yaml:"brokers" env:"KAFKA_BROKERS" env-default:"localhost:9092"`
	ConsumerGroup string   `yaml:"consumer_group" env:"KAFKA_GROUP" env-default:"stitching-group"`
	InputTopic    string   `yaml:"input_topic" env:"KAFKA_STITCHING_TOPIC" env-default:"stitching-tasks"`
	ResultTopic   string   `yaml:"result_topic" env:"KAFKA_RESULT_TOPIC" env-default:"render-results"`
}

type MinioConfig struct {
	Endpoint     string `yaml:"endpoint" env:"S3_ENDPOINT" env-default:"localhost:9000"`
	AccessKey    string `yaml:"access_key" env:"S3_ACCESS_KEY" env-required:"true"`
	SecretKey    string `yaml:"secret_key" env:"S3_SECRET_KEY" env-required:"true"`
	Bucket       string `yaml:"bucket" env:"S3_BUCKET" env-default:"projects"`
	UseSSL       bool   `yaml:"use_ssl" env:"S3_USE_SSL" env-default:"false"`
	OutputBucket string `yaml:"output_bucket" env:"S3_OUTPUT_BUCKET" env-default:"results"`
}

func Load(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = os.Getenv("CONFIG_PATH")
		if configPath == "" {
			configPath = "config/config.yaml"
		}
	}

	// Check if file exists, if not, try to read from env only
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		var cfg Config
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			return nil, fmt.Errorf("cannot read config from env: %w", err)
		}
		return &cfg, nil
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

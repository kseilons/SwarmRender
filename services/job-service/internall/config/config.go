package config

import (
	"fmt"
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env      string           `yaml:"env" env:"ENV" env-default:"local"`
	Http     HttpServerConfig `yaml:"http"`
	Database DatabaseConfig   `yaml:"database"`
	Kafka    KafkaConfig      `yaml:"kafka"`
	Minio    MinioConfig      `yaml:"minio"`
}

type HttpServerConfig struct {
	Address    string `yaml:"address" env:"HTTP_ADDRESS" env-default:"0.0.0.0:8081"`
	MaxWorkers int    `yaml:"max_workers" env:"MAX_WORKERS" env-default:"10"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host" env:"DB_HOST" env-default:"localhost"`
	Port     int    `yaml:"port" env:"DB_PORT" env-default:"5432"`
	User     string `yaml:"user" env:"DB_USER" env-required:"true"`
	Password string `yaml:"password" env:"DB_PASSWORD" env-required:"true"`
	Name     string `yaml:"name" env:"DB_NAME" env-required:"true"`
	SSLMode  string `yaml:"sslmode" env:"DB_SSLMODE" env-default:"disable"`
}

func (c DatabaseConfig) URL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Name, c.SSLMode)
}

type KafkaConfig struct {
	Brokers        []string `yaml:"brokers" env:"KAFKA_BROKERS" env-default:"localhost:9092"`
	RenderTopic    string   `yaml:"render_topic" env:"KAFKA_RENDER_TOPIC" env-default:"render-tasks"`
	MetadataTopic  string   `yaml:"metadata_topic" env:"KAFKA_METADATA_TOPIC" env-default:"metadata-tasks"`
	StitchingTopic string   `yaml:"stitching_topic" env:"KAFKA_STITCHING_TOPIC" env-default:"stitching-tasks"`
	ResultTopic    string   `yaml:"result_topic" env:"KAFKA_RESULT_TOPIC" env-default:"render-results"`
	ResultGroupID  string   `yaml:"result_group_id" env:"KAFKA_RESULT_GROUP_ID" env-default:"job-service-results"`
}

type MinioConfig struct {
	Endpoint        string `yaml:"endpoint" env:"MINIO_ENDPOINT" env-default:"localhost:9000"`
	AccessKey       string `yaml:"access_key" env:"MINIO_ACCESS_KEY" env-default:"admin"`
	SecretKey       string `yaml:"secret_key" env:"MINIO_SECRET_KEY" env-default:"password"`
	UseSSL          bool   `yaml:"use_ssl" env:"MINIO_USE_SSL" env-default:"false"`
	Bucket          string `yaml:"bucket" env:"MINIO_BUCKET" env-default:"projects"`
	PresignedExpiry int    `yaml:"presigned_expiry" env:"MINIO_PRESIGNED_EXPIRY" env-default:"3600"`
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

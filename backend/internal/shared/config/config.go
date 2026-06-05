package config

import (
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	AppName        string `env:"APP_NAME" env-default:"koran-ai"`
	AppEnv         string `env:"APP_ENV" env-default:"development"`
	AppPort        int    `env:"APP_PORT" env-default:"8080"`
	DBHost         string `env:"DB_HOST" env-default:"localhost"`
	DBPort         int    `env:"DB_PORT" env-default:"5432"`
	DBUser         string `env:"DB_USER" env-default:"postgres"`
	DBPassword     string `env:"DB_PASSWORD" env-default:"postgres"`
	DBName         string `env:"DB_NAME" env-default:"koran_ai_prod"`
	DBSSLMode      string `env:"DB_SSLMODE" env-default:"disable"`
	RedisHost      string `env:"REDIS_HOST" env-default:"localhost"`
	RedisPort      int    `env:"REDIS_PORT" env-default:"6379"`
	RedisPassword  string `env:"REDIS_PASSWORD" env-default:""`
	LogLevel       string `env:"LOG_LEVEL" env-default:"debug"`
	InternalAPIKey string `env:"INTERNAL_API_KEY" env-default:"KoranAISecretKey2026!"`
}

// Load loads the configuration from .env file or environment variables.
func Load(envFilePath string) (*Config, error) {
	var cfg Config

	// Check if the file exists
	if envFilePath != "" {
		if _, err := os.Stat(envFilePath); err == nil {
			err = cleanenv.ReadConfig(envFilePath, &cfg)
			if err != nil {
				return nil, err
			}
			return &cfg, nil
		}
	}

	// Fallback to loading directly from system env
	err := cleanenv.ReadEnv(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Global variable config is avoided as per coding standards ("No Global Variable")
// We will pass Config around via dependency injection.
func InitConfig() *Config {
	cfg, err := Load(".env")
	if err != nil {
		log.Printf("Warning: Failed to load .env file, loading from system environment: %v", err)
		cfg, err = Load("")
		if err != nil {
			log.Fatalf("Error loading configurations: %v", err)
		}
	}
	return cfg
}

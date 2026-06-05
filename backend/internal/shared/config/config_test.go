package config

import (
	"os"
	"testing"
)

func TestLoad_WithFile(t *testing.T) {
	tempFile := "test_config.env"
	content := `
APP_NAME=test-app
APP_ENV=testing
APP_PORT=9090
DB_HOST=127.0.0.1
DB_PORT=5433
DB_USER=test_user
DB_PASSWORD=test_pass
DB_NAME=test_db
DB_SSLMODE=require
REDIS_HOST=127.0.0.1
REDIS_PORT=6380
REDIS_PASSWORD=redis_pass
LOG_LEVEL=info
INTERNAL_API_KEY=test_key
`
	err := os.WriteFile(tempFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp env file: %v", err)
	}
	defer os.Remove(tempFile)

	cfg, err := Load(tempFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.AppName != "test-app" {
		t.Errorf("Expected AppName 'test-app', got %s", cfg.AppName)
	}
	if cfg.AppEnv != "testing" {
		t.Errorf("Expected AppEnv 'testing', got %s", cfg.AppEnv)
	}
	if cfg.AppPort != 9090 {
		t.Errorf("Expected AppPort 9090, got %d", cfg.AppPort)
	}
	if cfg.DBHost != "127.0.0.1" {
		t.Errorf("Expected DBHost '127.0.0.1', got %s", cfg.DBHost)
	}
	if cfg.DBPort != 5433 {
		t.Errorf("Expected DBPort 5433, got %d", cfg.DBPort)
	}
	if cfg.DBUser != "test_user" {
		t.Errorf("Expected DBUser 'test_user', got %s", cfg.DBUser)
	}
	if cfg.DBPassword != "test_pass" {
		t.Errorf("Expected DBPassword 'test_pass', got %s", cfg.DBPassword)
	}
	if cfg.DBName != "test_db" {
		t.Errorf("Expected DBName 'test_db', got %s", cfg.DBName)
	}
	if cfg.DBSSLMode != "require" {
		t.Errorf("Expected DBSSLMode 'require', got %s", cfg.DBSSLMode)
	}
	if cfg.RedisHost != "127.0.0.1" {
		t.Errorf("Expected RedisHost '127.0.0.1', got %s", cfg.RedisHost)
	}
	if cfg.RedisPort != 6380 {
		t.Errorf("Expected RedisPort 6380, got %d", cfg.RedisPort)
	}
	if cfg.RedisPassword != "redis_pass" {
		t.Errorf("Expected RedisPassword 'redis_pass', got %s", cfg.RedisPassword)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("Expected LogLevel 'info', got %s", cfg.LogLevel)
	}
	if cfg.InternalAPIKey != "test_key" {
		t.Errorf("Expected InternalAPIKey 'test_key', got %s", cfg.InternalAPIKey)
	}
}

func TestLoad_FromSystemEnv(t *testing.T) {
	os.Setenv("APP_NAME", "sys-app")
	os.Setenv("APP_PORT", "1234")
	defer func() {
		os.Unsetenv("APP_NAME")
		os.Unsetenv("APP_PORT")
	}()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Failed to load system env: %v", err)
	}

	if cfg.AppName != "sys-app" {
		t.Errorf("Expected AppName 'sys-app', got %s", cfg.AppName)
	}
	if cfg.AppPort != 1234 {
		t.Errorf("Expected AppPort 1234, got %d", cfg.AppPort)
	}
}

func TestLoad_Defaults(t *testing.T) {
	// Clear any env vars that might interfere
	envVars := []string{
		"APP_NAME", "APP_ENV", "APP_PORT",
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE",
		"REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD",
		"LOG_LEVEL", "INTERNAL_API_KEY",
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Failed to load with defaults: %v", err)
	}

	if cfg.AppName != "koran-ai" {
		t.Errorf("Expected default AppName 'koran-ai', got %s", cfg.AppName)
	}
	if cfg.AppPort != 8080 {
		t.Errorf("Expected default AppPort 8080, got %d", cfg.AppPort)
	}
	if cfg.DBPort != 5432 {
		t.Errorf("Expected default DBPort 5432, got %d", cfg.DBPort)
	}
	if cfg.RedisPort != 6379 {
		t.Errorf("Expected default RedisPort 6379, got %d", cfg.RedisPort)
	}
}

package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	JWT      JWTConfig
	Database DatabaseConfig
	Logging  LoggingConfig
}

type AppConfig struct {
	Environment string
	Port        string
	FilesDir    string
}

type JWTConfig struct {
	Secret          string
	ExpirationHours int
}

type DatabaseConfig struct {
	Path string
}

type LoggingConfig struct {
	Level string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		App: AppConfig{
			Environment: getEnv("APP_ENV", "development"),
			Port:        getEnv("PORT", "8080"),
			FilesDir:    getEnv("FILES_DIR", "./files"),
		},
		JWT: JWTConfig{
			Secret:          getEnv("JWT_SECRET", ""),
			ExpirationHours: getEnvAsInt("JWT_EXPIRATION_HOURS", 24),
		},
		Database: DatabaseConfig{
			Path: getEnv("DB_PATH", "./data.db"),
		},
		Logging: LoggingConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}

	if len(c.JWT.Secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters long")
	}

	filesDir, err := os.Open(c.App.FilesDir)
	if err != nil {
		return fmt.Errorf("%v does not exist: %w", c.App.FilesDir, err)
	}

	info, err := filesDir.Stat()
	if err != nil {
		return fmt.Errorf("failed to read file %v: %w", c.App.FilesDir, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%v is not a directory, exiting", c.App.FilesDir)
	}

	return nil
}

func (c *Config) IsDevelopment() bool {
	return c.App.Environment == "development"
}

func (c *Config) IsProduction() bool {
	return c.App.Environment == "production"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

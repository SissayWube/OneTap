package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	Server    ServerConfig
	Auth      AuthConfig
	RateLimit RateLimitConfig
	Logging   LoggingConfig
	Files     FilesConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port           int
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	MaxRequestSize int64
}

// AuthConfig holds authentication-related configuration
type AuthConfig struct {
	JWTSecret     string
	JWTExpiration time.Duration
	BcryptCost    int
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	MaxAttempts   int
	TimeWindow    time.Duration
	BlockDuration time.Duration
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string
	Format string
}

// FilesConfig holds file paths configuration
type FilesConfig struct {
	CustomersPath       string
	TransactionsPath    string
	SampleCustomersPath string
}

// Load loads configuration from environment variables with defaults
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:           getEnvAsInt("SERVER_PORT", 8080),
			ReadTimeout:    getEnvAsDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout:   getEnvAsDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			MaxRequestSize: getEnvAsInt64("SERVER_MAX_REQUEST_SIZE", 10485760), // 10MB
		},
		Auth: AuthConfig{
			JWTSecret:     getEnv("JWT_SECRET", ""),
			JWTExpiration: getEnvAsDuration("JWT_EXPIRATION", 24*time.Hour),
			BcryptCost:    getEnvAsInt("BCRYPT_COST", 10),
		},
		RateLimit: RateLimitConfig{
			MaxAttempts:   getEnvAsInt("RATE_LIMIT_MAX_ATTEMPTS", 5),
			TimeWindow:    getEnvAsDuration("RATE_LIMIT_TIME_WINDOW", 15*time.Minute),
			BlockDuration: getEnvAsDuration("RATE_LIMIT_BLOCK_DURATION", 15*time.Minute),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		Files: FilesConfig{
			CustomersPath:       getEnv("CUSTOMERS_FILE", "data/customers.json"),
			TransactionsPath:    getEnv("TRANSACTIONS_FILE", "data/transactions.json"),
			SampleCustomersPath: getEnv("SAMPLE_CUSTOMERS_FILE", "data/sample_customers.csv"),
		},
	}

	// Validate required configuration
	if cfg.Auth.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
	}

	if cfg.Auth.BcryptCost < 10 {
		return nil, fmt.Errorf("BCRYPT_COST must be at least 10")
	}

	return cfg, nil
}

// Helper functions to read environment variables with defaults

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

func getEnvAsInt64(key string, defaultValue int64) int64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

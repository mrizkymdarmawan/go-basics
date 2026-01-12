// Package config handles application configuration.
// It follows the 12-Factor App methodology where configuration
// is loaded from environment variables with sensible defaults.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration.
// We use a struct to group related settings together,
// making it easy to pass configuration through the application.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	// Port is the HTTP port the server listens on.
	Port string

	// ReadTimeout is the maximum duration for reading the entire request.
	// This prevents slow clients from holding connections open.
	ReadTimeout time.Duration

	// WriteTimeout is the maximum duration for writing the response.
	// This prevents slow clients from holding connections open.
	WriteTimeout time.Duration

	// IdleTimeout is the maximum time to wait for the next request
	// when keep-alives are enabled.
	IdleTimeout time.Duration
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	// DSN is the Data Source Name (connection string) for MySQL.
	// Format: user:password@tcp(host:port)/dbname?parseTime=true
	DSN string

	// MaxOpenConns is the maximum number of open connections to the database.
	// Setting this too high can exhaust database resources.
	// Setting this too low can cause connection contention.
	MaxOpenConns int

	// MaxIdleConns is the maximum number of idle connections in the pool.
	// Should be less than or equal to MaxOpenConns.
	MaxIdleConns int

	// ConnMaxLifetime is the maximum time a connection can be reused.
	// Helps with load balancing and handling database restarts.
	ConnMaxLifetime time.Duration
}

// JWTConfig holds JWT (JSON Web Token) authentication settings.
type JWTConfig struct {
	// Secret is the key used to sign JWT tokens.
	// IMPORTANT: In production, use a strong, random secret (at least 32 bytes).
	// Never commit the actual secret to version control.
	Secret string

	// AccessTokenDuration is how long an access token is valid.
	// Keep this short (15-30 minutes) for security.
	// Users will need to refresh tokens or re-login after expiration.
	AccessTokenDuration time.Duration

	// Issuer identifies who created the token.
	// Useful when you have multiple services issuing tokens.
	Issuer string
}

// Load reads configuration from environment variables with defaults.
// This is the preferred pattern because:
// 1. Environment variables are easy to change in different environments
// 2. Secrets don't get committed to version control
// 3. Works well with Docker, Kubernetes, and cloud platforms
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			// getEnv is a helper that returns a default if the env var is empty
			Port:         getEnv("SERVER_PORT", "8080"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 5*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:  getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
		Database: DatabaseConfig{
			DSN:             getEnv("DB_DSN", "root:root@tcp(localhost:3306)/db_go_basics?parseTime=true"),
			MaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 10),
			MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", 30*time.Minute),
		},
		JWT: JWTConfig{
			// IMPORTANT: Change this secret in production!
			// Use: openssl rand -base64 32
			Secret:              getEnv("JWT_SECRET", "your-256-bit-secret-key-change-in-production"),
			AccessTokenDuration: getDurationEnv("JWT_ACCESS_TOKEN_DURATION", 15*time.Minute),
			Issuer:              getEnv("JWT_ISSUER", "go-basics"),
		},
	}
}

// getEnv returns the value of an environment variable or a default value.
// This is a common pattern in Go applications.
func getEnv(key, defaultValue string) string {
	// os.Getenv returns empty string if the variable is not set
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getIntEnv returns an integer from an environment variable or a default.
// We use strconv.Atoi to convert string to int.
func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		// Atoi = "ASCII to Integer"
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getDurationEnv returns a time.Duration from an environment variable.
// Duration strings can be like "5s", "10m", "1h30m".
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		// ParseDuration understands "ns", "us", "ms", "s", "m", "h"
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

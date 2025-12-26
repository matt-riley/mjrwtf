package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	// Database configuration
	DatabaseURL string

	// Server configuration
	ServerPort int
	BaseURL    string

	// CORS configuration
	AllowedOrigins string

	// Authentication
	AuthToken string

	// Session configuration
	SecureCookies bool // Set to true in production with HTTPS

	// Discord webhook configuration
	DiscordWebhookURL string

	// GeoIP configuration
	GeoIPEnabled  bool
	GeoIPDatabase string

	// Logging configuration
	LogLevel  string // debug, info, warn, error (default: info)
	LogFormat string // json, pretty (default: json)

	// Metrics configuration
	MetricsAuthEnabled bool // Enable authentication for /metrics endpoint (default: false)

	// Database operation timeout configuration
	DBTimeout time.Duration // Timeout for database operations (default: 5s)
}

// LoadConfig loads configuration from environment variables and .env file
// It returns an error if required configuration is missing or invalid
func LoadConfig() (*Config, error) {
	// Try to load .env file (ignore error if file doesn't exist)
	_ = godotenv.Load()

	config := &Config{
		DatabaseURL:        getEnv("DATABASE_URL", ""),
		ServerPort:         getEnvAsInt("SERVER_PORT", 8080),
		BaseURL:            getEnv("BASE_URL", "http://localhost:8080"),
		AllowedOrigins:     getEnv("ALLOWED_ORIGINS", "*"),
		AuthToken:          getEnv("AUTH_TOKEN", ""),
		SecureCookies:      getEnvAsBool("SECURE_COOKIES", false),
		DiscordWebhookURL:  getEnv("DISCORD_WEBHOOK_URL", ""),
		GeoIPEnabled:       getEnvAsBool("GEOIP_ENABLED", false),
		GeoIPDatabase:      getEnv("GEOIP_DATABASE", ""),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		LogFormat:          getEnv("LOG_FORMAT", "json"),
		MetricsAuthEnabled: getEnvAsBool("METRICS_AUTH_ENABLED", false),
		DBTimeout:          getEnvAsDuration("DB_TIMEOUT", 5*time.Second),
	}

	// Validate required configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate checks that required configuration values are present
func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	if c.AuthToken == "" {
		return fmt.Errorf("AUTH_TOKEN is required")
	}

	if c.ServerPort < 1 || c.ServerPort > 65535 {
		return fmt.Errorf("SERVER_PORT must be between 1 and 65535")
	}

	// If GeoIP is enabled, database path is required
	if c.GeoIPEnabled && c.GeoIPDatabase == "" {
		return fmt.Errorf("GEOIP_DATABASE is required when GEOIP_ENABLED is true")
	}

	return nil
}

// getEnv gets an environment variable with a fallback default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as an integer with a fallback default value
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

// getEnvAsBool gets an environment variable as a boolean with a fallback default value
func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

// getEnvAsDuration gets an environment variable as a duration with a fallback default value
// Duration should be specified as a string like "5s", "100ms", "1m30s"
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

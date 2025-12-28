package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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

	// Rate limiting configuration
	RedirectRateLimitPerMinute int
	APIRateLimitPerMinute      int

	// Redirect click recording configuration
	RedirectClickWorkers   int // Worker goroutines for async click recording (default: 100)
	RedirectClickQueueSize int // Queue size for async click recording (default: RedirectClickWorkers*2)

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

	// Security headers configuration
	EnableHSTS bool // Enable Strict-Transport-Security header (default: false, only enable when behind TLS)

	// Database operation timeout configuration
	DBTimeout time.Duration // Timeout for database operations (default: 5s)
}

// LoadConfig loads configuration from environment variables and .env file
// It returns an error if required configuration is missing or invalid
func LoadConfig() (*Config, error) {
	// Try to load .env file (ignore error if file doesn't exist)
	_ = godotenv.Load()

	serverPort, err := getEnvAsInt("SERVER_PORT", 8080)
	if err != nil {
		return nil, err
	}
	secureCookies, err := getEnvAsBool("SECURE_COOKIES", false)
	if err != nil {
		return nil, err
	}
	redirectRateLimitPerMinute, err := getEnvAsInt("REDIRECT_RATE_LIMIT_PER_MINUTE", 120)
	if err != nil {
		return nil, err
	}
	apiRateLimitPerMinute, err := getEnvAsInt("API_RATE_LIMIT_PER_MINUTE", 60)
	if err != nil {
		return nil, err
	}
	redirectClickWorkers, err := getEnvAsInt("REDIRECT_CLICK_WORKERS", 100)
	if err != nil {
		return nil, err
	}
	redirectClickQueueSize, err := getEnvAsInt("REDIRECT_CLICK_QUEUE_SIZE", redirectClickWorkers*2)
	if err != nil {
		return nil, err
	}
	geoIPEnabled, err := getEnvAsBool("GEOIP_ENABLED", false)
	if err != nil {
		return nil, err
	}
	metricsAuthEnabled, err := getEnvAsBool("METRICS_AUTH_ENABLED", false)
	if err != nil {
		return nil, err
	}
	enableHSTS, err := getEnvAsBool("ENABLE_HSTS", false)
	if err != nil {
		return nil, err
	}
	dbTimeout, err := getEnvAsDuration("DB_TIMEOUT", 5*time.Second)
	if err != nil {
		return nil, err
	}

	config := &Config{
		DatabaseURL:                getEnv("DATABASE_URL", ""),
		ServerPort:                 serverPort,
		BaseURL:                    getEnv("BASE_URL", "http://localhost:8080"),
		AllowedOrigins:             getEnv("ALLOWED_ORIGINS", "*"),
		AuthToken:                  getEnv("AUTH_TOKEN", ""),
		SecureCookies:              secureCookies,
		RedirectRateLimitPerMinute: redirectRateLimitPerMinute,
		APIRateLimitPerMinute:      apiRateLimitPerMinute,
		RedirectClickWorkers:       redirectClickWorkers,
		RedirectClickQueueSize:     redirectClickQueueSize,
		DiscordWebhookURL:          getEnv("DISCORD_WEBHOOK_URL", ""),
		GeoIPEnabled:               geoIPEnabled,
		GeoIPDatabase:              getEnv("GEOIP_DATABASE", ""),
		LogLevel:                   getEnv("LOG_LEVEL", "info"),
		LogFormat:                  getEnv("LOG_FORMAT", "json"),
		MetricsAuthEnabled:         metricsAuthEnabled,
		EnableHSTS:                 enableHSTS,
		DBTimeout:                  dbTimeout,
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
		return ErrMissingDatabaseURL
	}

	// SQLite-only: reject URL-form connection strings (e.g. network URLs),
	// to avoid sqlite creating a local file literally named after the URL.
	if strings.Contains(c.DatabaseURL, "://") {
		scheme, _, _ := strings.Cut(strings.ToLower(c.DatabaseURL), "://")
		return fmt.Errorf("%w %q (SQLite file paths only)", ErrUnsupportedDatabaseURLScheme, scheme)
	}

	if c.AuthToken == "" {
		return ErrMissingAuthToken
	}

	if c.ServerPort < 1 || c.ServerPort > 65535 {
		return ErrInvalidServerPortRange
	}

	if c.RedirectRateLimitPerMinute < 1 {
		return ErrInvalidRedirectRateLimit
	}

	if c.APIRateLimitPerMinute < 1 {
		return ErrInvalidAPIRateLimit
	}

	if c.RedirectClickWorkers < 1 {
		return ErrInvalidRedirectClickWorkers
	}

	if c.RedirectClickQueueSize < 1 {
		return ErrInvalidRedirectClickQueueSize
	}

	// If GeoIP is enabled, database path is required
	if c.GeoIPEnabled && c.GeoIPDatabase == "" {
		return ErrMissingGeoIPDatabase
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

// getEnvAsInt gets an environment variable as an integer.
// Defaults apply only when the env var is unset.
func getEnvAsInt(key string, defaultValue int) (int, error) {
	valueStr, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue, nil
	}
	if valueStr == "" {
		return 0, fmt.Errorf("%s %w", key, ErrEnvVarEmpty)
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, fmt.Errorf("%s %w, got %q", key, ErrEnvVarNotInt, valueStr)
	}

	return value, nil
}

// getEnvAsBool gets an environment variable as a boolean.
// Defaults apply only when the env var is unset.
func getEnvAsBool(key string, defaultValue bool) (bool, error) {
	valueStr, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue, nil
	}
	if valueStr == "" {
		return false, fmt.Errorf("%s %w", key, ErrEnvVarEmpty)
	}

	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return false, fmt.Errorf("%s %w, got %q", key, ErrEnvVarNotBool, valueStr)
	}

	return value, nil
}

// getEnvAsDuration gets an environment variable as a duration.
// Defaults apply only when the env var is unset.
// Duration should be specified as a string like "5s", "100ms", "1m30s".
func getEnvAsDuration(key string, defaultValue time.Duration) (time.Duration, error) {
	valueStr, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue, nil
	}
	if valueStr == "" {
		return 0, fmt.Errorf("%s %w", key, ErrEnvVarEmpty)
	}

	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return 0, fmt.Errorf("%s %w, got %q", key, ErrEnvVarNotDuration, valueStr)
	}

	return value, nil
}

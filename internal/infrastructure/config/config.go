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
	// AuthToken is kept for backward compatibility (legacy AUTH_TOKEN).
	// Prefer AuthTokens/ActiveAuthTokens for validation.
	AuthToken  string
	AuthTokens []string

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

	// URL status checker configuration
	URLStatusCheckerEnabled                bool
	URLStatusCheckerPollInterval           time.Duration
	URLStatusCheckerAliveRecheckInterval   time.Duration
	URLStatusCheckerGoneRecheckInterval    time.Duration
	URLStatusCheckerBatchSize              int
	URLStatusCheckerConcurrency            int
	URLStatusCheckerArchiveLookupEnabled   bool
	URLStatusCheckerArchiveRecheckInterval time.Duration

	// Tailscale configuration
	TailscaleEnabled       bool   // Enable Tailscale tsnet server (default: false)
	TailscaleHostname      string // Hostname to register with Tailscale (required when enabled)
	TailscaleAuthKey       string // Tailscale auth key for automatic authentication (optional)
	TailscaleStateDir      string // Directory for Tailscale state storage (required when enabled)
	TailscaleFunnelEnabled bool   // Enable Tailscale Funnel for public HTTPS (default: false)
	TailscaleControlURL    string // Custom Tailscale control plane URL (optional, for headscale)
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

	urlStatusCheckerEnabled, err := getEnvAsBool("URL_STATUS_CHECKER_ENABLED", false)
	if err != nil {
		return nil, err
	}
	urlStatusCheckerPollInterval, err := getEnvAsDuration("URL_STATUS_CHECKER_POLL_INTERVAL", 5*time.Minute)
	if err != nil {
		return nil, err
	}
	urlStatusCheckerAliveRecheckInterval, err := getEnvAsDuration("URL_STATUS_CHECKER_ALIVE_RECHECK_INTERVAL", 6*time.Hour)
	if err != nil {
		return nil, err
	}
	urlStatusCheckerGoneRecheckInterval, err := getEnvAsDuration("URL_STATUS_CHECKER_GONE_RECHECK_INTERVAL", 24*time.Hour)
	if err != nil {
		return nil, err
	}
	urlStatusCheckerBatchSize, err := getEnvAsInt("URL_STATUS_CHECKER_BATCH_SIZE", 100)
	if err != nil {
		return nil, err
	}
	urlStatusCheckerConcurrency, err := getEnvAsInt("URL_STATUS_CHECKER_CONCURRENCY", 5)
	if err != nil {
		return nil, err
	}
	urlStatusCheckerArchiveLookupEnabled, err := getEnvAsBool("URL_STATUS_CHECKER_ARCHIVE_LOOKUP_ENABLED", true)
	if err != nil {
		return nil, err
	}
	urlStatusCheckerArchiveRecheckInterval, err := getEnvAsDuration("URL_STATUS_CHECKER_ARCHIVE_RECHECK_INTERVAL", 7*24*time.Hour)
	if err != nil {
		return nil, err
	}

	// Tailscale configuration
	tailscaleEnabled, err := getEnvAsBool("TAILSCALE_ENABLED", false)
	if err != nil {
		return nil, err
	}
	tailscaleFunnelEnabled, err := getEnvAsBool("TAILSCALE_FUNNEL_ENABLED", false)
	if err != nil {
		return nil, err
	}

	authTokens, err := getEnvAuthTokens()
	if err != nil {
		return nil, err
	}

	config := &Config{
		DatabaseURL:                getEnv("DATABASE_URL", ""),
		ServerPort:                 serverPort,
		BaseURL:                    getEnv("BASE_URL", "http://localhost:8080"),
		AllowedOrigins:             getEnv("ALLOWED_ORIGINS", "*"),
		AuthToken:                  authTokens[0],
		AuthTokens:                 authTokens,
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

		URLStatusCheckerEnabled:                urlStatusCheckerEnabled,
		URLStatusCheckerPollInterval:           urlStatusCheckerPollInterval,
		URLStatusCheckerAliveRecheckInterval:   urlStatusCheckerAliveRecheckInterval,
		URLStatusCheckerGoneRecheckInterval:    urlStatusCheckerGoneRecheckInterval,
		URLStatusCheckerBatchSize:              urlStatusCheckerBatchSize,
		URLStatusCheckerConcurrency:            urlStatusCheckerConcurrency,
		URLStatusCheckerArchiveLookupEnabled:   urlStatusCheckerArchiveLookupEnabled,
		URLStatusCheckerArchiveRecheckInterval: urlStatusCheckerArchiveRecheckInterval,

		TailscaleEnabled:       tailscaleEnabled,
		TailscaleHostname:      getEnv("TAILSCALE_HOSTNAME", ""),
		TailscaleAuthKey:       getEnv("TAILSCALE_AUTH_KEY", ""),
		TailscaleStateDir:      getEnv("TAILSCALE_STATE_DIR", ""),
		TailscaleFunnelEnabled: tailscaleFunnelEnabled,
		TailscaleControlURL:    getEnv("TAILSCALE_CONTROL_URL", ""),
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
		return fmt.Errorf("%w: DATABASE_URL scheme=%q (SQLite file paths only)", ErrUnsupportedDatabaseURLScheme, scheme)
	}

	if len(c.ActiveAuthTokens()) == 0 {
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

	if c.URLStatusCheckerEnabled {
		if c.URLStatusCheckerPollInterval <= 0 || c.URLStatusCheckerAliveRecheckInterval <= 0 || c.URLStatusCheckerGoneRecheckInterval <= 0 {
			return fmt.Errorf("URL status checker intervals (poll, alive recheck, gone recheck) must be > 0")
		}
		if c.URLStatusCheckerBatchSize < 1 || c.URLStatusCheckerConcurrency < 1 {
			return fmt.Errorf("URL status checker batch size and concurrency must be > 0")
		}
	}

	if c.URLStatusCheckerEnabled && c.URLStatusCheckerArchiveLookupEnabled && c.URLStatusCheckerArchiveRecheckInterval <= 0 {
		return fmt.Errorf("archive recheck interval must be > 0 when archive lookup is enabled")
	}

	// If GeoIP is enabled, database path is required
	if c.GeoIPEnabled && c.GeoIPDatabase == "" {
		return ErrMissingGeoIPDatabase
	}

	// Tailscale validation: hostname and state dir required when enabled
	if c.TailscaleEnabled {
		if c.TailscaleHostname == "" {
			return ErrMissingTailscaleHostname
		}
		if c.TailscaleStateDir == "" {
			return ErrMissingTailscaleStateDir
		}
	}

	return nil
}

// ActiveAuthTokens returns the set of currently-active authentication tokens.
//
// Backward compatibility:
//   - If AuthTokens is populated, it is used.
//   - Otherwise, if AuthToken is set, it is returned as a single-element slice.
func (c *Config) ActiveAuthTokens() []string {
	if len(c.AuthTokens) > 0 {
		return c.AuthTokens
	}
	if c.AuthToken != "" {
		return []string{c.AuthToken}
	}
	return nil
}

func getEnvAuthTokens() ([]string, error) {
	if raw, ok := os.LookupEnv("AUTH_TOKENS"); ok {
		parts := strings.Split(raw, ",")
		tokens := make([]string, 0, len(parts))
		seen := make(map[string]struct{}, len(parts))
		for _, p := range parts {
			t := strings.TrimSpace(p)
			if t == "" {
				continue
			}
			if _, ok := seen[t]; ok {
				continue
			}
			seen[t] = struct{}{}
			tokens = append(tokens, t)
		}
		if len(tokens) == 0 {
			return nil, ErrMissingAuthToken
		}
		return tokens, nil
	}

	// Legacy single-token config.
	if token := getEnv("AUTH_TOKEN", ""); token != "" {
		return []string{token}, nil
	}
	return nil, ErrMissingAuthToken
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
		return 0, fmt.Errorf("%w: %s", ErrEnvVarEmpty, key)
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, fmt.Errorf("%w: %s (got %q)", ErrEnvVarNotInt, key, valueStr)
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
		return false, fmt.Errorf("%w: %s", ErrEnvVarEmpty, key)
	}

	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return false, fmt.Errorf("%w: %s (got %q)", ErrEnvVarNotBool, key, valueStr)
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
		return 0, fmt.Errorf("%w: %s", ErrEnvVarEmpty, key)
	}

	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return 0, fmt.Errorf("%w: %s (got %q)", ErrEnvVarNotDuration, key, valueStr)
	}

	return value, nil
}

package config

import (
	"errors"
	"os"
	"testing"
	"time"
)

func TestLoadConfig_Success(t *testing.T) {
	// Set required environment variables
	os.Unsetenv("AUTH_TOKENS")
	os.Setenv("DATABASE_URL", "./database.db")
	os.Setenv("AUTH_TOKEN", "test-token")
	defer cleanEnv()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if config.DatabaseURL != "./database.db" {
		t.Errorf("Expected DATABASE_URL to be './database.db', got: %s", config.DatabaseURL)
	}

	if config.AuthToken != "test-token" {
		t.Errorf("Expected AUTH_TOKEN to be 'test-token', got: %s", config.AuthToken)
	}
	if len(config.AuthTokens) != 1 || config.AuthTokens[0] != "test-token" {
		t.Errorf("Expected AuthTokens to be ['test-token'], got: %#v", config.AuthTokens)
	}

	// Check default value for ServerPort
	if config.ServerPort != 8080 {
		t.Errorf("Expected default ServerPort to be 8080, got: %d", config.ServerPort)
	}
}

func TestLoadConfig_RejectsURLScheme(t *testing.T) {
	os.Setenv("DATABASE_URL", "unsupported://example")
	os.Setenv("AUTH_TOKEN", "test-token")
	defer cleanEnv()

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("Expected error for URL-form DATABASE_URL, got nil")
	}
	if !errors.Is(err, ErrUnsupportedDatabaseURLScheme) {
		t.Fatalf("Expected ErrUnsupportedDatabaseURLScheme, got: %v", err)
	}
}

func TestLoadConfig_MissingDatabaseURL(t *testing.T) {
	// Set only AUTH_TOKEN, leave DATABASE_URL empty
	os.Setenv("AUTH_TOKEN", "test-token")
	os.Unsetenv("DATABASE_URL")
	defer cleanEnv()

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("Expected error for missing DATABASE_URL, got nil")
	}

	if !errors.Is(err, ErrMissingDatabaseURL) {
		t.Fatalf("Expected ErrMissingDatabaseURL, got: %v", err)
	}
}

func TestLoadConfig_MissingAuthToken(t *testing.T) {
	// Set only DATABASE_URL, leave auth tokens empty
	os.Setenv("DATABASE_URL", "./database.db")
	os.Unsetenv("AUTH_TOKEN")
	os.Unsetenv("AUTH_TOKENS")
	defer cleanEnv()

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("Expected error for missing auth token(s), got nil")
	}

	if !errors.Is(err, ErrMissingAuthToken) {
		t.Fatalf("Expected ErrMissingAuthToken, got: %v", err)
	}
}

func TestLoadConfig_AuthTokens_Multiple(t *testing.T) {
	os.Setenv("DATABASE_URL", "./database.db")
	os.Setenv("AUTH_TOKENS", "tokenA,tokenB")
	defer cleanEnv()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.AuthToken != "tokenA" {
		t.Fatalf("Expected AuthToken to be 'tokenA' (first token), got: %q", cfg.AuthToken)
	}
	if len(cfg.AuthTokens) != 2 || cfg.AuthTokens[0] != "tokenA" || cfg.AuthTokens[1] != "tokenB" {
		t.Fatalf("Expected AuthTokens to be ['tokenA','tokenB'], got: %#v", cfg.AuthTokens)
	}
}

func TestLoadConfig_AuthTokens_EmptyOrMalformed(t *testing.T) {
	os.Setenv("DATABASE_URL", "./database.db")
	os.Setenv("AUTH_TOKENS", ",   ,")
	defer cleanEnv()

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("Expected error for empty AUTH_TOKENS, got nil")
	}
	if !errors.Is(err, ErrMissingAuthToken) {
		t.Fatalf("Expected ErrMissingAuthToken, got: %v", err)
	}
}

func TestLoadConfig_AuthTokens_IgnoresEmptyEntries(t *testing.T) {
	os.Setenv("DATABASE_URL", "./database.db")
	os.Setenv("AUTH_TOKENS", "tokenA,, tokenB,  ")
	defer cleanEnv()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(cfg.AuthTokens) != 2 || cfg.AuthTokens[0] != "tokenA" || cfg.AuthTokens[1] != "tokenB" {
		t.Fatalf("Expected AuthTokens to be ['tokenA','tokenB'], got: %#v", cfg.AuthTokens)
	}
}

func TestLoadConfig_AuthTokens_TakesPrecedenceOverAuthToken(t *testing.T) {
	os.Setenv("DATABASE_URL", "./database.db")
	os.Setenv("AUTH_TOKEN", "legacy")
	os.Setenv("AUTH_TOKENS", "tokenA,tokenB")
	defer cleanEnv()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if cfg.AuthToken != "tokenA" {
		t.Fatalf("Expected AuthToken to be first AUTH_TOKENS entry 'tokenA', got: %q", cfg.AuthToken)
	}
	if len(cfg.AuthTokens) != 2 || cfg.AuthTokens[0] != "tokenA" || cfg.AuthTokens[1] != "tokenB" {
		t.Fatalf("Expected AuthTokens to be ['tokenA','tokenB'], got: %#v", cfg.AuthTokens)
	}
}

func TestLoadConfig_AuthTokens_Deduplicates(t *testing.T) {
	os.Setenv("DATABASE_URL", "./database.db")
	os.Setenv("AUTH_TOKENS", "tokenA,tokenA,tokenB,tokenA")
	defer cleanEnv()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(cfg.AuthTokens) != 2 || cfg.AuthTokens[0] != "tokenA" || cfg.AuthTokens[1] != "tokenB" {
		t.Fatalf("Expected AuthTokens to be ['tokenA','tokenB'], got: %#v", cfg.AuthTokens)
	}
}

func TestConfig_ActiveAuthTokens(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want []string
	}{
		{
			name: "uses AuthTokens when populated",
			cfg:  &Config{AuthTokens: []string{"a", "b"}, AuthToken: "legacy"},
			want: []string{"a", "b"},
		},
		{
			name: "falls back to AuthToken when AuthTokens empty",
			cfg:  &Config{AuthToken: "a"},
			want: []string{"a"},
		},
		{
			name: "nil when both empty",
			cfg:  &Config{},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.ActiveAuthTokens()
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d (got=%#v)", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestLoadConfig_CustomServerPort(t *testing.T) {
	// Set required environment variables with custom port
	os.Setenv("DATABASE_URL", "./database.db")
	os.Setenv("AUTH_TOKEN", "test-token")
	os.Setenv("SERVER_PORT", "9000")
	defer cleanEnv()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if config.ServerPort != 9000 {
		t.Errorf("Expected ServerPort to be 9000, got: %d", config.ServerPort)
	}
}

func TestLoadConfig_InvalidServerPortString(t *testing.T) {
	os.Setenv("DATABASE_URL", "./database.db")
	os.Setenv("AUTH_TOKEN", "test-token")
	os.Setenv("SERVER_PORT", "abc")
	defer cleanEnv()

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("Expected error for invalid SERVER_PORT, got nil")
	}

	if !errors.Is(err, ErrEnvVarNotInt) {
		t.Fatalf("Expected ErrEnvVarNotInt, got: %v", err)
	}
}

func TestLoadConfig_InvalidServerPort(t *testing.T) {
	// Set required environment variables with invalid port
	os.Setenv("DATABASE_URL", "./database.db")
	os.Setenv("AUTH_TOKEN", "test-token")
	os.Setenv("SERVER_PORT", "70000")
	defer cleanEnv()

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("Expected error for invalid SERVER_PORT, got nil")
	}

	if !errors.Is(err, ErrInvalidServerPortRange) {
		t.Fatalf("Expected ErrInvalidServerPortRange, got: %v", err)
	}
}

func TestLoadConfig_DiscordWebhookURL(t *testing.T) {
	// Set required environment variables with Discord webhook
	os.Setenv("DATABASE_URL", "./database.db")
	os.Setenv("AUTH_TOKEN", "test-token")
	os.Setenv("DISCORD_WEBHOOK_URL", "https://discord.com/api/webhooks/123/abc")
	defer cleanEnv()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if config.DiscordWebhookURL != "https://discord.com/api/webhooks/123/abc" {
		t.Errorf("Expected DiscordWebhookURL to be set, got: %s", config.DiscordWebhookURL)
	}
}

func TestLoadConfig_GeoIPEnabled(t *testing.T) {
	// Set required environment variables with GeoIP enabled
	os.Setenv("DATABASE_URL", "./database.db")
	os.Setenv("AUTH_TOKEN", "test-token")
	os.Setenv("GEOIP_ENABLED", "true")
	os.Setenv("GEOIP_DATABASE", "/path/to/geoip.db")
	defer cleanEnv()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !config.GeoIPEnabled {
		t.Error("Expected GeoIPEnabled to be true")
	}

	if config.GeoIPDatabase != "/path/to/geoip.db" {
		t.Errorf("Expected GeoIPDatabase to be '/path/to/geoip.db', got: %s", config.GeoIPDatabase)
	}
}

func TestLoadConfig_GeoIPEnabledWithoutDatabase(t *testing.T) {
	// Set required environment variables with GeoIP enabled but no database
	os.Setenv("DATABASE_URL", "./database.db")
	os.Setenv("AUTH_TOKEN", "test-token")
	os.Setenv("GEOIP_ENABLED", "true")
	os.Unsetenv("GEOIP_DATABASE")
	defer cleanEnv()

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("Expected error for GEOIP_ENABLED without GEOIP_DATABASE, got nil")
	}

	if !errors.Is(err, ErrMissingGeoIPDatabase) {
		t.Fatalf("Expected ErrMissingGeoIPDatabase, got: %v", err)
	}
}

func TestLoadConfig_DefaultValues(t *testing.T) {
	// Set only required environment variables
	os.Setenv("DATABASE_URL", "./database.db")
	os.Setenv("AUTH_TOKEN", "test-token")
	defer cleanEnv()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check default values
	if config.ServerPort != 8080 {
		t.Errorf("Expected default ServerPort to be 8080, got: %d", config.ServerPort)
	}

	if config.RedirectRateLimitPerMinute != 120 {
		t.Errorf("Expected default RedirectRateLimitPerMinute to be 120, got: %d", config.RedirectRateLimitPerMinute)
	}

	if config.APIRateLimitPerMinute != 60 {
		t.Errorf("Expected default APIRateLimitPerMinute to be 60, got: %d", config.APIRateLimitPerMinute)
	}

	if config.RedirectClickWorkers != 100 {
		t.Errorf("Expected default RedirectClickWorkers to be 100, got: %d", config.RedirectClickWorkers)
	}

	if config.RedirectClickQueueSize != 200 {
		t.Errorf("Expected default RedirectClickQueueSize to be 200, got: %d", config.RedirectClickQueueSize)
	}

	if config.DiscordWebhookURL != "" {
		t.Errorf("Expected default DiscordWebhookURL to be empty, got: %s", config.DiscordWebhookURL)
	}

	if config.GeoIPEnabled {
		t.Error("Expected default GeoIPEnabled to be false")
	}

	if config.GeoIPDatabase != "" {
		t.Errorf("Expected default GeoIPDatabase to be empty, got: %s", config.GeoIPDatabase)
	}
}

func TestLoadConfig_CustomRateLimits(t *testing.T) {
	os.Setenv("DATABASE_URL", "./database.db")
	os.Setenv("AUTH_TOKEN", "test-token")
	os.Setenv("REDIRECT_RATE_LIMIT_PER_MINUTE", "10")
	os.Setenv("API_RATE_LIMIT_PER_MINUTE", "5")
	defer cleanEnv()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if config.RedirectRateLimitPerMinute != 10 {
		t.Errorf("Expected RedirectRateLimitPerMinute to be 10, got: %d", config.RedirectRateLimitPerMinute)
	}

	if config.APIRateLimitPerMinute != 5 {
		t.Errorf("Expected APIRateLimitPerMinute to be 5, got: %d", config.APIRateLimitPerMinute)
	}
}

func TestLoadConfig_InvalidRateLimits(t *testing.T) {
	os.Setenv("DATABASE_URL", "./database.db")
	os.Setenv("AUTH_TOKEN", "test-token")
	os.Setenv("REDIRECT_RATE_LIMIT_PER_MINUTE", "0")
	os.Setenv("API_RATE_LIMIT_PER_MINUTE", "-1")
	defer cleanEnv()

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("Expected error for invalid rate limits, got nil")
	}

	if !errors.Is(err, ErrInvalidRedirectRateLimit) {
		t.Fatalf("Expected ErrInvalidRedirectRateLimit, got: %v", err)
	}
}

func TestLoadConfig_RateLimitsInvalidStringsFallbackToDefaults(t *testing.T) {
	os.Setenv("DATABASE_URL", "./database.db")
	os.Setenv("AUTH_TOKEN", "test-token")
	os.Setenv("REDIRECT_RATE_LIMIT_PER_MINUTE", "abc")
	os.Setenv("API_RATE_LIMIT_PER_MINUTE", "def")
	defer cleanEnv()

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("Expected error for invalid rate limit values, got nil")
	}

	if !errors.Is(err, ErrEnvVarNotInt) {
		t.Fatalf("Expected ErrEnvVarNotInt, got: %v", err)
	}
}

func TestLoadConfig_InvalidRedirectClickWorkers(t *testing.T) {
	os.Setenv("DATABASE_URL", "./database.db")
	os.Setenv("AUTH_TOKEN", "test-token")
	os.Setenv("REDIRECT_CLICK_WORKERS", "0")
	defer cleanEnv()

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("Expected error for invalid REDIRECT_CLICK_WORKERS, got nil")
	}

	if !errors.Is(err, ErrInvalidRedirectClickWorkers) {
		t.Fatalf("Expected ErrInvalidRedirectClickWorkers, got: %v", err)
	}
}

func TestLoadConfig_InvalidRedirectClickQueueSize(t *testing.T) {
	os.Setenv("DATABASE_URL", "./database.db")
	os.Setenv("AUTH_TOKEN", "test-token")
	os.Setenv("REDIRECT_CLICK_QUEUE_SIZE", "0")
	defer cleanEnv()

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("Expected error for invalid REDIRECT_CLICK_QUEUE_SIZE, got nil")
	}

	if !errors.Is(err, ErrInvalidRedirectClickQueueSize) {
		t.Fatalf("Expected ErrInvalidRedirectClickQueueSize, got: %v", err)
	}
}

func TestLoadConfig_RedirectClickConfigInvalidStrings(t *testing.T) {
	os.Setenv("DATABASE_URL", "./database.db")
	os.Setenv("AUTH_TOKEN", "test-token")
	os.Setenv("REDIRECT_CLICK_WORKERS", "abc")
	defer cleanEnv()

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("Expected error for invalid redirect click config values, got nil")
	}

	if !errors.Is(err, ErrEnvVarNotInt) {
		t.Fatalf("Expected ErrEnvVarNotInt, got: %v", err)
	}
}

func TestGetEnvAsInt_InvalidValue(t *testing.T) {
	os.Setenv("TEST_INT", "not-a-number")
	defer os.Unsetenv("TEST_INT")

	_, err := getEnvAsInt("TEST_INT", 42)
	if err == nil {
		t.Fatal("Expected error for invalid int, got nil")
	}

	if !errors.Is(err, ErrEnvVarNotInt) {
		t.Fatalf("Expected ErrEnvVarNotInt, got: %v", err)
	}
}

func TestGetEnvAsInt_EmptyString(t *testing.T) {
	os.Setenv("TEST_INT", "")
	defer os.Unsetenv("TEST_INT")

	_, err := getEnvAsInt("TEST_INT", 42)
	if err == nil {
		t.Fatal("Expected error for empty int env var, got nil")
	}

	if !errors.Is(err, ErrEnvVarEmpty) {
		t.Fatalf("Expected ErrEnvVarEmpty, got: %v", err)
	}
}

func TestGetEnvAsBool_InvalidValue(t *testing.T) {
	os.Setenv("TEST_BOOL", "not-a-bool")
	defer os.Unsetenv("TEST_BOOL")

	_, err := getEnvAsBool("TEST_BOOL", true)
	if err == nil {
		t.Fatal("Expected error for invalid bool, got nil")
	}

	if !errors.Is(err, ErrEnvVarNotBool) {
		t.Fatalf("Expected ErrEnvVarNotBool, got: %v", err)
	}
}

func TestGetEnvAsBool_EmptyString(t *testing.T) {
	os.Setenv("TEST_BOOL", "")
	defer os.Unsetenv("TEST_BOOL")

	_, err := getEnvAsBool("TEST_BOOL", true)
	if err == nil {
		t.Fatal("Expected error for empty bool env var, got nil")
	}

	if !errors.Is(err, ErrEnvVarEmpty) {
		t.Fatalf("Expected ErrEnvVarEmpty, got: %v", err)
	}
}

func TestGetEnvAsBool_ValidValues(t *testing.T) {
	testCases := []struct {
		value    string
		expected bool
	}{
		{"true", true},
		{"false", false},
		{"1", true},
		{"0", false},
		{"t", true},
		{"f", false},
		{"T", true},
		{"F", false},
		{"TRUE", true},
		{"FALSE", false},
	}

	for _, tc := range testCases {
		os.Setenv("TEST_BOOL", tc.value)
		result, err := getEnvAsBool("TEST_BOOL", false)
		if err != nil {
			t.Fatalf("Unexpected error for value '%s': %v", tc.value, err)
		}
		if result != tc.expected {
			t.Errorf("For value '%s', expected %v, got %v", tc.value, tc.expected, result)
		}
		os.Unsetenv("TEST_BOOL")
	}
}

func TestGetEnvAsDuration_InvalidValue(t *testing.T) {
	os.Setenv("TEST_DURATION", "not-a-duration")
	defer os.Unsetenv("TEST_DURATION")

	_, err := getEnvAsDuration("TEST_DURATION", 5*time.Second)
	if err == nil {
		t.Fatal("Expected error for invalid duration, got nil")
	}

	if !errors.Is(err, ErrEnvVarNotDuration) {
		t.Fatalf("Expected ErrEnvVarNotDuration, got: %v", err)
	}
}

func TestGetEnvAsDuration_EmptyString(t *testing.T) {
	os.Setenv("TEST_DURATION", "")
	defer os.Unsetenv("TEST_DURATION")

	_, err := getEnvAsDuration("TEST_DURATION", 5*time.Second)
	if err == nil {
		t.Fatal("Expected error for empty duration env var, got nil")
	}

	if !errors.Is(err, ErrEnvVarEmpty) {
		t.Fatalf("Expected ErrEnvVarEmpty, got: %v", err)
	}
}

// cleanEnv clears all environment variables used in tests
func cleanEnv() {
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("SERVER_PORT")
	os.Unsetenv("AUTH_TOKEN")
	os.Unsetenv("AUTH_TOKENS")
	os.Unsetenv("DISCORD_WEBHOOK_URL")
	os.Unsetenv("GEOIP_ENABLED")
	os.Unsetenv("GEOIP_DATABASE")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("LOG_FORMAT")
	os.Unsetenv("METRICS_AUTH_ENABLED")
	os.Unsetenv("REDIRECT_RATE_LIMIT_PER_MINUTE")
	os.Unsetenv("API_RATE_LIMIT_PER_MINUTE")
	os.Unsetenv("REDIRECT_CLICK_WORKERS")
	os.Unsetenv("REDIRECT_CLICK_QUEUE_SIZE")
	os.Unsetenv("DB_TIMEOUT")
	os.Unsetenv("URL_STATUS_CHECKER_ENABLED")
	os.Unsetenv("URL_STATUS_CHECKER_POLL_INTERVAL")
	os.Unsetenv("URL_STATUS_CHECKER_ALIVE_RECHECK_INTERVAL")
	os.Unsetenv("URL_STATUS_CHECKER_GONE_RECHECK_INTERVAL")
	os.Unsetenv("URL_STATUS_CHECKER_BATCH_SIZE")
	os.Unsetenv("URL_STATUS_CHECKER_CONCURRENCY")
	os.Unsetenv("URL_STATUS_CHECKER_ARCHIVE_LOOKUP_ENABLED")
	os.Unsetenv("URL_STATUS_CHECKER_ARCHIVE_RECHECK_INTERVAL")
}

func TestLoadConfig_LoggingDefaults(t *testing.T) {
	os.Setenv("DATABASE_URL", "./database.db")
	os.Setenv("AUTH_TOKEN", "test-token")
	defer cleanEnv()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if config.LogLevel != "info" {
		t.Errorf("Expected default LogLevel 'info', got: %s", config.LogLevel)
	}
	if config.LogFormat != "json" {
		t.Errorf("Expected default LogFormat 'json', got: %s", config.LogFormat)
	}
}

func TestLoadConfig_CustomLogLevel(t *testing.T) {
	os.Setenv("DATABASE_URL", "./database.db")
	os.Setenv("AUTH_TOKEN", "test-token")
	os.Setenv("LOG_LEVEL", "debug")
	defer cleanEnv()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if config.LogLevel != "debug" {
		t.Errorf("Expected LogLevel 'debug', got: %s", config.LogLevel)
	}
}

func TestLoadConfig_CustomLogFormat(t *testing.T) {
	os.Setenv("DATABASE_URL", "./database.db")
	os.Setenv("AUTH_TOKEN", "test-token")
	os.Setenv("LOG_FORMAT", "pretty")
	defer cleanEnv()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if config.LogFormat != "pretty" {
		t.Errorf("Expected LogFormat 'pretty', got: %s", config.LogFormat)
	}
}

func TestLoadConfig_MetricsAuthEnabled(t *testing.T) {
	os.Setenv("DATABASE_URL", "./database.db")
	os.Setenv("AUTH_TOKEN", "test-token")
	os.Setenv("METRICS_AUTH_ENABLED", "true")
	defer cleanEnv()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !config.MetricsAuthEnabled {
		t.Error("Expected MetricsAuthEnabled to be true")
	}
}

func TestLoadConfig_MetricsAuthDefaultFalse(t *testing.T) {
	os.Setenv("DATABASE_URL", "./database.db")
	os.Setenv("AUTH_TOKEN", "test-token")
	defer cleanEnv()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if config.MetricsAuthEnabled {
		t.Error("Expected default MetricsAuthEnabled to be false")
	}
}

func TestLoadConfig_DBTimeoutDefault(t *testing.T) {
	os.Setenv("DATABASE_URL", "./database.db")
	os.Setenv("AUTH_TOKEN", "test-token")
	defer cleanEnv()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expectedTimeout := 5 * time.Second
	if config.DBTimeout != expectedTimeout {
		t.Errorf("Expected default DBTimeout to be %v, got: %v", expectedTimeout, config.DBTimeout)
	}
}

func TestLoadConfig_CustomDBTimeout(t *testing.T) {
	os.Setenv("DATABASE_URL", "./database.db")
	os.Setenv("AUTH_TOKEN", "test-token")
	os.Setenv("DB_TIMEOUT", "10s")
	defer cleanEnv()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expectedTimeout := 10 * time.Second
	if config.DBTimeout != expectedTimeout {
		t.Errorf("Expected DBTimeout to be %v, got: %v", expectedTimeout, config.DBTimeout)
	}
}

func TestLoadConfig_InvalidDBTimeout(t *testing.T) {
	os.Setenv("DATABASE_URL", "./database.db")
	os.Setenv("AUTH_TOKEN", "test-token")
	os.Setenv("DB_TIMEOUT", "invalid")
	defer cleanEnv()

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("Expected error for invalid DB_TIMEOUT, got nil")
	}

	if !errors.Is(err, ErrEnvVarNotDuration) {
		t.Fatalf("Expected ErrEnvVarNotDuration, got: %v", err)
	}
}

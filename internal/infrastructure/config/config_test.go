package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig_Success(t *testing.T) {
	// Set required environment variables
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

	expectedError := "DATABASE_URL is required"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got: %s", expectedError, err.Error())
	}
}

func TestLoadConfig_MissingAuthToken(t *testing.T) {
	// Set only DATABASE_URL, leave AUTH_TOKEN empty
	os.Setenv("DATABASE_URL", "./database.db")
	os.Unsetenv("AUTH_TOKEN")
	defer cleanEnv()

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("Expected error for missing AUTH_TOKEN, got nil")
	}

	expectedError := "AUTH_TOKEN is required"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got: %s", expectedError, err.Error())
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

	expectedError := "SERVER_PORT must be an integer, got \"abc\""
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got: %s", expectedError, err.Error())
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

	expectedError := "SERVER_PORT must be between 1 and 65535"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got: %s", expectedError, err.Error())
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

	expectedError := "GEOIP_DATABASE is required when GEOIP_ENABLED is true"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got: %s", expectedError, err.Error())
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

	expectedError := "REDIRECT_RATE_LIMIT_PER_MINUTE must be greater than 0"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got: %s", expectedError, err.Error())
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

	expectedError := "REDIRECT_RATE_LIMIT_PER_MINUTE must be an integer, got \"abc\""
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got: %s", expectedError, err.Error())
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

	expectedError := "REDIRECT_CLICK_WORKERS must be greater than 0"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got: %s", expectedError, err.Error())
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

	expectedError := "REDIRECT_CLICK_QUEUE_SIZE must be greater than 0"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got: %s", expectedError, err.Error())
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

	expectedError := "REDIRECT_CLICK_WORKERS must be an integer, got \"abc\""
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got: %s", expectedError, err.Error())
	}
}

func TestGetEnvAsInt_InvalidValue(t *testing.T) {
	os.Setenv("TEST_INT", "not-a-number")
	defer os.Unsetenv("TEST_INT")

	_, err := getEnvAsInt("TEST_INT", 42)
	if err == nil {
		t.Fatal("Expected error for invalid int, got nil")
	}

	expectedError := "TEST_INT must be an integer, got \"not-a-number\""
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got: %s", expectedError, err.Error())
	}
}

func TestGetEnvAsInt_EmptyString(t *testing.T) {
	os.Setenv("TEST_INT", "")
	defer os.Unsetenv("TEST_INT")

	_, err := getEnvAsInt("TEST_INT", 42)
	if err == nil {
		t.Fatal("Expected error for empty int env var, got nil")
	}

	expectedError := "TEST_INT must not be empty"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got: %s", expectedError, err.Error())
	}
}

func TestGetEnvAsBool_InvalidValue(t *testing.T) {
	os.Setenv("TEST_BOOL", "not-a-bool")
	defer os.Unsetenv("TEST_BOOL")

	_, err := getEnvAsBool("TEST_BOOL", true)
	if err == nil {
		t.Fatal("Expected error for invalid bool, got nil")
	}

	expectedError := "TEST_BOOL must be a boolean, got \"not-a-bool\""
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got: %s", expectedError, err.Error())
	}
}

func TestGetEnvAsBool_EmptyString(t *testing.T) {
	os.Setenv("TEST_BOOL", "")
	defer os.Unsetenv("TEST_BOOL")

	_, err := getEnvAsBool("TEST_BOOL", true)
	if err == nil {
		t.Fatal("Expected error for empty bool env var, got nil")
	}

	expectedError := "TEST_BOOL must not be empty"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got: %s", expectedError, err.Error())
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

	expectedError := "TEST_DURATION must be a duration, got \"not-a-duration\""
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got: %s", expectedError, err.Error())
	}
}

func TestGetEnvAsDuration_EmptyString(t *testing.T) {
	os.Setenv("TEST_DURATION", "")
	defer os.Unsetenv("TEST_DURATION")

	_, err := getEnvAsDuration("TEST_DURATION", 5*time.Second)
	if err == nil {
		t.Fatal("Expected error for empty duration env var, got nil")
	}

	expectedError := "TEST_DURATION must not be empty"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got: %s", expectedError, err.Error())
	}
}

// cleanEnv clears all environment variables used in tests
func cleanEnv() {
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("SERVER_PORT")
	os.Unsetenv("AUTH_TOKEN")
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

	expectedError := "DB_TIMEOUT must be a duration, got \"invalid\""
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got: %s", expectedError, err.Error())
	}
}

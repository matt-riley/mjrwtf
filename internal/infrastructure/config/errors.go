package config

import "errors"

var (
	// ErrMissingDatabaseURL is returned when DATABASE_URL is not set or is empty.
	ErrMissingDatabaseURL = errors.New("DATABASE_URL is required")
	// ErrUnsupportedDatabaseURLScheme is returned when DATABASE_URL is URL-form (contains a scheme).
	ErrUnsupportedDatabaseURLScheme = errors.New("DATABASE_URL has unsupported scheme")
	// ErrMissingAuthToken is returned when AUTH_TOKEN is not set or is empty.
	ErrMissingAuthToken = errors.New("AUTH_TOKEN is required")
	// ErrInvalidServerPortRange is returned when SERVER_PORT is outside the range 1..65535.
	ErrInvalidServerPortRange = errors.New("SERVER_PORT must be between 1 and 65535")
	// ErrInvalidRedirectRateLimit is returned when REDIRECT_RATE_LIMIT_PER_MINUTE is < 1.
	ErrInvalidRedirectRateLimit = errors.New("REDIRECT_RATE_LIMIT_PER_MINUTE must be greater than 0")
	// ErrInvalidAPIRateLimit is returned when API_RATE_LIMIT_PER_MINUTE is < 1.
	ErrInvalidAPIRateLimit = errors.New("API_RATE_LIMIT_PER_MINUTE must be greater than 0")
	// ErrInvalidRedirectClickWorkers is returned when REDIRECT_CLICK_WORKERS is < 1.
	ErrInvalidRedirectClickWorkers = errors.New("REDIRECT_CLICK_WORKERS must be greater than 0")
	// ErrInvalidRedirectClickQueueSize is returned when REDIRECT_CLICK_QUEUE_SIZE is < 1.
	ErrInvalidRedirectClickQueueSize = errors.New("REDIRECT_CLICK_QUEUE_SIZE must be greater than 0")
	// ErrMissingGeoIPDatabase is returned when GEOIP_ENABLED is true but GEOIP_DATABASE is not set.
	ErrMissingGeoIPDatabase = errors.New("GEOIP_DATABASE is required when GEOIP_ENABLED is true")

	// ErrEnvVarEmpty is wrapped when an env var is present but empty.
	ErrEnvVarEmpty = errors.New("must not be empty")
	// ErrEnvVarNotInt is wrapped when an env var cannot be parsed as an integer.
	ErrEnvVarNotInt = errors.New("must be an integer")
	// ErrEnvVarNotBool is wrapped when an env var cannot be parsed as a boolean.
	ErrEnvVarNotBool = errors.New("must be a boolean")
	// ErrEnvVarNotDuration is wrapped when an env var cannot be parsed as a duration.
	ErrEnvVarNotDuration = errors.New("must be a duration")
)

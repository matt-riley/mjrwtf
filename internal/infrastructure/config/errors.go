package config

import "errors"

var (
	ErrMissingDatabaseURL            = errors.New("DATABASE_URL is required")
	ErrUnsupportedDatabaseURLScheme  = errors.New("DATABASE_URL has unsupported scheme")
	ErrMissingAuthToken              = errors.New("AUTH_TOKEN is required")
	ErrInvalidServerPortRange        = errors.New("SERVER_PORT must be between 1 and 65535")
	ErrInvalidRedirectRateLimit      = errors.New("REDIRECT_RATE_LIMIT_PER_MINUTE must be greater than 0")
	ErrInvalidAPIRateLimit           = errors.New("API_RATE_LIMIT_PER_MINUTE must be greater than 0")
	ErrInvalidRedirectClickWorkers   = errors.New("REDIRECT_CLICK_WORKERS must be greater than 0")
	ErrInvalidRedirectClickQueueSize = errors.New("REDIRECT_CLICK_QUEUE_SIZE must be greater than 0")
	ErrMissingGeoIPDatabase          = errors.New("GEOIP_DATABASE is required when GEOIP_ENABLED is true")

	ErrEnvVarEmpty       = errors.New("must not be empty")
	ErrEnvVarNotInt      = errors.New("must be an integer")
	ErrEnvVarNotBool     = errors.New("must be a boolean")
	ErrEnvVarNotDuration = errors.New("must be a duration")
)

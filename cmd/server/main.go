package main

import (
	"context"
	"database/sql"
	"os"
	"os/signal"
	"strings"
	"syscall"

	_ "github.com/lib/pq"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/config"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/http/server"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/logging"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		// Use a basic logger for startup errors before full config is loaded
		logger := logging.New("error", "json")
		logger.Fatal().Err(err).Msg("failed to load configuration")
	}

	// Initialize logger with configured level and format
	logger := logging.New(cfg.LogLevel, cfg.LogFormat)
	logger.Info().
		Str("log_level", cfg.LogLevel).
		Str("log_format", cfg.LogFormat).
		Int("port", cfg.ServerPort).
		Msg("configuration loaded")

	// Open database connection
	db, err := openDatabase(cfg.DatabaseURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to open database")
	}
	defer db.Close()

	logger.Info().Msg("database connection established")

	// Create server
	srv, err := server.New(cfg, db, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create server")
	}

	// Channel to listen for errors from the server
	serverErrors := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		serverErrors <- srv.Start()
	}()

	// Channel to listen for interrupt or terminate signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a signal or an error
	select {
	case err := <-serverErrors:
		logger.Fatal().Err(err).Msg("server error")
	case sig := <-shutdown:
		logger.Info().Str("signal", sig.String()).Msg("received shutdown signal")

		// Graceful shutdown with timeout context
		// Match server's ShutdownTimeout to ensure process eventually terminates
		ctx, cancel := context.WithTimeout(context.Background(), server.ShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Fatal().Err(err).Msg("error during shutdown")
		}

		logger.Info().Msg("server shutdown complete")
	}
}

// openDatabase opens a database connection based on the connection string
func openDatabase(dbURL string) (*sql.DB, error) {
	// Determine database driver based on URL
	driver := "sqlite3"
	if strings.HasPrefix(dbURL, "postgres://") || strings.HasPrefix(dbURL, "postgresql://") {
		driver = "postgres"
	}

	db, err := sql.Open(driver, dbURL)
	if err != nil {
		return nil, err
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

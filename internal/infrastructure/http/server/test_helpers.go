package server

import (
	"database/sql"
	"testing"
	"time"

	"github.com/matt-riley/mjrwtf/internal/infrastructure/config"
	"github.com/matt-riley/mjrwtf/internal/migrations"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(tb testing.TB) *sql.DB {
	tb.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		tb.Fatalf("failed to open test database: %v", err)
	}

	// For in-memory SQLite, limit to 1 connection to serialize writes
	// This prevents SQLITE_BUSY errors during concurrent operations
	db.SetMaxOpenConns(1)

	// Set SQLite dialect and use embedded migrations
	if err := goose.SetDialect("sqlite3"); err != nil {
		db.Close()
		tb.Fatalf("failed to set goose dialect: %v", err)
	}
	goose.SetBaseFS(migrations.SQLiteMigrations)

	if err := goose.Up(db, migrations.SQLiteDir); err != nil {
		db.Close()
		tb.Fatalf("failed to run migrations: %v", err)
	}

	return db
}

// testConfig creates a test configuration with sensible defaults
func testConfig() *config.Config {
	return &config.Config{
		ServerPort:     8080,
		BaseURL:        "http://localhost:8080",
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
		DBTimeout:      5 * time.Second,
	}
}

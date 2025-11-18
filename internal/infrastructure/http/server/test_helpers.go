package server

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/matt-riley/mjrwtf/internal/migrations"
	"github.com/pressly/goose/v3"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(tb testing.TB) *sql.DB {
	tb.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		tb.Fatalf("failed to open test database: %v", err)
	}

	// Set SQLite dialect and use embedded migrations
	goose.SetDialect("sqlite3")
	goose.SetBaseFS(migrations.SQLiteMigrations)
	
	if err := goose.Up(db, migrations.SQLiteDir); err != nil {
		db.Close()
		tb.Fatalf("failed to run migrations: %v", err)
	}

	return db
}

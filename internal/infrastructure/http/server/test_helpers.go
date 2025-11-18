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

// verifyTablesExist checks that required tables exist in the database
func verifyTablesExist(tb testing.TB, db *sql.DB) {
tb.Helper()

rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
if err != nil {
tb.Fatalf("failed to query tables: %v", err)
}
defer rows.Close()

tables := make(map[string]bool)
for rows.Next() {
var name string
if err := rows.Scan(&name); err != nil {
tb.Fatalf("failed to scan table name: %v", err)
}
tables[name] = true
}

tb.Logf("Tables found: %v", tables)

if !tables["urls"] {
tb.Fatal("urls table not found")
}
if !tables["clicks"] {
tb.Fatal("clicks table not found")
}
}

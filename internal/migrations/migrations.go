package migrations

import (
	"embed"
)

// SQLiteMigrations contains embedded SQLite migration files
//
//go:embed sqlite/*.sql
var SQLiteMigrations embed.FS

// PostgresMigrations contains embedded PostgreSQL migration files
//
//go:embed postgres/*.sql
var PostgresMigrations embed.FS

const (
	// SQLiteDir is the directory path for SQLite migrations
	SQLiteDir = "sqlite"
	// PostgresDir is the directory path for PostgreSQL migrations
	PostgresDir = "postgres"
)

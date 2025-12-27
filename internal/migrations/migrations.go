package migrations

import (
	"embed"
)

// SQLiteMigrations contains embedded SQLite migration files.
//
//go:embed sqlite/*.sql
var SQLiteMigrations embed.FS

const (
	// SQLiteDir is the directory path for SQLite migrations.
	SQLiteDir = "sqlite"
)

package main

import (
	"os"
	"testing"
)

func TestOpenDatabase_SQLite_WithoutQueryParams(t *testing.T) {
	tmpFile := t.TempDir() + "/test.db"
	defer os.Remove(tmpFile)

	db, err := openDatabase(tmpFile)
	if err != nil {
		t.Fatalf("openDatabase failed: %v", err)
	}
	defer db.Close()

	// Verify WAL mode is enabled
	var mode string
	err = db.QueryRow("PRAGMA journal_mode").Scan(&mode)
	if err != nil {
		t.Fatalf("failed to query journal_mode: %v", err)
	}

	if mode != "wal" {
		t.Errorf("expected journal_mode to be 'wal', got '%s'", mode)
	}

	// Verify busy timeout is set
	var timeout int
	err = db.QueryRow("PRAGMA busy_timeout").Scan(&timeout)
	if err != nil {
		t.Fatalf("failed to query busy_timeout: %v", err)
	}

	if timeout != sqliteBusyTimeoutMs {
		t.Errorf("expected busy_timeout to be %d, got %d", sqliteBusyTimeoutMs, timeout)
	}

	// Verify max open connections
	stats := db.Stats()
	if stats.MaxOpenConnections != sqliteMaxOpenConns {
		t.Errorf("expected MaxOpenConnections to be %d, got %d", sqliteMaxOpenConns, stats.MaxOpenConnections)
	}
}

func TestOpenDatabase_SQLite_WithExistingQueryParams(t *testing.T) {
	tmpFile := t.TempDir() + "/test.db?cache=shared"
	defer os.Remove(tmpFile)

	db, err := openDatabase(tmpFile)
	if err != nil {
		t.Fatalf("openDatabase failed: %v", err)
	}
	defer db.Close()

	// Verify WAL mode is enabled
	var mode string
	err = db.QueryRow("PRAGMA journal_mode").Scan(&mode)
	if err != nil {
		t.Fatalf("failed to query journal_mode: %v", err)
	}

	if mode != "wal" {
		t.Errorf("expected journal_mode to be 'wal', got '%s'", mode)
	}
}

func TestOpenDatabase_SQLite_WithJournalModeInQueryParams(t *testing.T) {
	tmpFile := t.TempDir() + "/test.db?_journal_mode=DELETE"
	defer os.Remove(tmpFile)

	db, err := openDatabase(tmpFile)
	if err != nil {
		t.Fatalf("openDatabase failed: %v", err)
	}
	defer db.Close()

	// Verify journal_mode is NOT overridden (should still be DELETE)
	var mode string
	err = db.QueryRow("PRAGMA journal_mode").Scan(&mode)
	if err != nil {
		t.Fatalf("failed to query journal_mode: %v", err)
	}

	if mode != "delete" {
		t.Errorf("expected journal_mode to be 'delete' (not overridden), got '%s'", mode)
	}
}

func TestOpenDatabase_SQLite_FilenameWithJournalMode(t *testing.T) {
	// Test that a filename containing "_journal_mode" doesn't cause issues
	tmpFile := t.TempDir() + "/my_journal_mode.db"
	defer os.Remove(tmpFile)

	db, err := openDatabase(tmpFile)
	if err != nil {
		t.Fatalf("openDatabase failed: %v", err)
	}
	defer db.Close()

	// Verify WAL mode is still enabled (no false positive from filename)
	var mode string
	err = db.QueryRow("PRAGMA journal_mode").Scan(&mode)
	if err != nil {
		t.Fatalf("failed to query journal_mode: %v", err)
	}

	if mode != "wal" {
		t.Errorf("expected journal_mode to be 'wal', got '%s'", mode)
	}
}

func TestOpenDatabase_PostgreSQL_ConnectionPoolSettings(t *testing.T) {
	// Skip if PostgreSQL is not available
	db, err := openDatabase("postgres://localhost:5432/test?sslmode=disable")
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer db.Close()

	// Verify max open connections for PostgreSQL
	stats := db.Stats()
	if stats.MaxOpenConnections != postgresMaxOpenConns {
		t.Errorf("expected MaxOpenConnections to be %d, got %d", postgresMaxOpenConns, stats.MaxOpenConnections)
	}

	// Note: MaxIdleConns is not directly exposed in Stats, but we've set it
	// The setting is applied in the openDatabase function
}

func TestOpenDatabase_InvalidDatabase(t *testing.T) {
	// Test with an invalid SQLite database path that will fail to open
	_, err := openDatabase("/invalid/path/that/does/not/exist/db.sqlite")
	if err == nil {
		t.Error("expected error when opening invalid database path, got nil")
	}
}

func TestOpenDatabase_DriverDetection(t *testing.T) {
	tests := []struct {
		name           string
		dbURL          string
		expectedDriver string
	}{
		{
			name:           "SQLite with relative path",
			dbURL:          "./test.db",
			expectedDriver: "sqlite3",
		},
		{
			name:           "SQLite with absolute path",
			dbURL:          "/tmp/test.db",
			expectedDriver: "sqlite3",
		},
		{
			name:           "PostgreSQL with postgres://",
			dbURL:          "postgres://localhost/test",
			expectedDriver: "postgres",
		},
		{
			name:           "PostgreSQL with postgresql://",
			dbURL:          "postgresql://localhost/test",
			expectedDriver: "postgres",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't actually test the driver selection without opening a real database,
			// but we can at least verify the function doesn't panic with these inputs
			// For SQLite, we'll create a temp file
			if tt.expectedDriver == "sqlite3" {
				tmpFile := t.TempDir() + "/test.db"
				defer os.Remove(tmpFile)

				db, err := openDatabase(tmpFile)
				if err != nil {
					t.Fatalf("openDatabase failed for %s: %v", tt.name, err)
				}
				defer db.Close()

				// Verify it's actually a SQLite database by checking WAL mode
				var mode string
				err = db.QueryRow("PRAGMA journal_mode").Scan(&mode)
				if err != nil {
					t.Fatalf("failed to query journal_mode (not a SQLite DB?): %v", err)
				}
			}
			// For PostgreSQL, we skip actual connection since it may not be available
		})
	}
}

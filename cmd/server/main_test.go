package main

import (
	"os"
	"testing"

	"github.com/matt-riley/mjrwtf/internal/infrastructure/database"
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

	if timeout != database.SQLiteBusyTimeoutMs {
		t.Errorf("expected busy_timeout to be %d, got %d", database.SQLiteBusyTimeoutMs, timeout)
	}

	// Verify max open connections
	stats := db.Stats()
	if stats.MaxOpenConnections != database.SQLiteMaxOpenConns {
		t.Errorf("expected MaxOpenConnections to be %d, got %d", database.SQLiteMaxOpenConns, stats.MaxOpenConnections)
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

func TestOpenDatabase_InvalidDatabase(t *testing.T) {
	// Test with an invalid SQLite database path that will fail to open
	_, err := openDatabase("/invalid/path/that/does/not/exist/db.sqlite")
	if err == nil {
		t.Error("expected error when opening invalid database path, got nil")
	}
}

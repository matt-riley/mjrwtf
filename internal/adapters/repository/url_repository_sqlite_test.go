package repository

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/matt-riley/mjrwtf/internal/domain/url"
	"github.com/matt-riley/mjrwtf/internal/migrations"
	"github.com/pressly/goose/v3"
)

func setupSQLiteTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	// Create temporary directory for test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// Enable foreign keys
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}

	// Run migrations
	goose.SetBaseFS(migrations.SQLiteMigrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("failed to set dialect: %v", err)
	}

	if err := goose.Up(db, migrations.SQLiteDir); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Return cleanup function
	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

func TestSQLiteURLRepository_Create(t *testing.T) {
	db, cleanup := setupSQLiteTestDB(t)
	defer cleanup()

	repo := NewSQLiteURLRepository(db)

	t.Run("successfully create URL", func(t *testing.T) {
		u, err := url.NewURL("test123", "https://example.com", "testuser")
		if err != nil {
			t.Fatalf("failed to create URL: %v", err)
		}

		err = repo.Create(u)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if u.ID == 0 {
			t.Error("Create() should set ID")
		}
	})

	t.Run("duplicate short code returns error", func(t *testing.T) {
		u1, _ := url.NewURL("duplicate", "https://example.com", "testuser")
		err := repo.Create(u1)
		if err != nil {
			t.Fatalf("first Create() error = %v", err)
		}

		u2, _ := url.NewURL("duplicate", "https://example2.com", "testuser")
		err = repo.Create(u2)
		if err != url.ErrDuplicateShortCode {
			t.Errorf("Create() with duplicate short code error = %v, want %v", err, url.ErrDuplicateShortCode)
		}
	})
}

func TestSQLiteURLRepository_FindByShortCode(t *testing.T) {
	db, cleanup := setupSQLiteTestDB(t)
	defer cleanup()

	repo := NewSQLiteURLRepository(db)

	t.Run("find existing URL", func(t *testing.T) {
		u, _ := url.NewURL("findme", "https://example.com", "testuser")
		err := repo.Create(u)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		found, err := repo.FindByShortCode("findme")
		if err != nil {
			t.Fatalf("FindByShortCode() error = %v", err)
		}

		if found.ShortCode != u.ShortCode {
			t.Errorf("FindByShortCode() ShortCode = %v, want %v", found.ShortCode, u.ShortCode)
		}
		if found.OriginalURL != u.OriginalURL {
			t.Errorf("FindByShortCode() OriginalURL = %v, want %v", found.OriginalURL, u.OriginalURL)
		}
		if found.CreatedBy != u.CreatedBy {
			t.Errorf("FindByShortCode() CreatedBy = %v, want %v", found.CreatedBy, u.CreatedBy)
		}
	})

	t.Run("URL not found returns error", func(t *testing.T) {
		_, err := repo.FindByShortCode("notfound")
		if err != url.ErrURLNotFound {
			t.Errorf("FindByShortCode() error = %v, want %v", err, url.ErrURLNotFound)
		}
	})
}

func TestSQLiteURLRepository_Delete(t *testing.T) {
	db, cleanup := setupSQLiteTestDB(t)
	defer cleanup()

	repo := NewSQLiteURLRepository(db)

	t.Run("delete existing URL", func(t *testing.T) {
		u, _ := url.NewURL("deleteme", "https://example.com", "testuser")
		err := repo.Create(u)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		err = repo.Delete("deleteme")
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// Verify it's deleted
		_, err = repo.FindByShortCode("deleteme")
		if err != url.ErrURLNotFound {
			t.Errorf("After Delete(), FindByShortCode() error = %v, want %v", err, url.ErrURLNotFound)
		}
	})

	t.Run("delete non-existent URL succeeds", func(t *testing.T) {
		// SQLite DELETE with no matching rows succeeds without error
		err := repo.Delete("nonexistent")
		if err != nil {
			t.Errorf("Delete() non-existent error = %v, want nil", err)
		}
	})
}

func TestSQLiteURLRepository_List(t *testing.T) {
	db, cleanup := setupSQLiteTestDB(t)
	defer cleanup()

	repo := NewSQLiteURLRepository(db)

	// Create test data
	urls := []struct {
		shortCode string
		createdBy string
	}{
		{"url1", "user1"},
		{"url2", "user1"},
		{"url3", "user2"},
		{"url4", "user1"},
	}

	for _, u := range urls {
		url, _ := url.NewURL(u.shortCode, fmt.Sprintf("https://example.com/%s", u.shortCode), u.createdBy)
		if err := repo.Create(url); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	t.Run("list all URLs", func(t *testing.T) {
		results, err := repo.List("", 0, 0)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(results) != 4 {
			t.Errorf("List() returned %d URLs, want 4", len(results))
		}

		// Should be ordered by created_at DESC
		if results[0].ShortCode != "url4" {
			t.Errorf("List() first URL = %v, want url4", results[0].ShortCode)
		}
	})

	t.Run("list by created_by", func(t *testing.T) {
		results, err := repo.List("user1", 0, 0)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(results) != 3 {
			t.Errorf("List() returned %d URLs, want 3", len(results))
		}

		for _, u := range results {
			if u.CreatedBy != "user1" {
				t.Errorf("List() returned URL with CreatedBy = %v, want user1", u.CreatedBy)
			}
		}
	})

	t.Run("list with limit", func(t *testing.T) {
		results, err := repo.List("", 2, 0)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(results) != 2 {
			t.Errorf("List() with limit=2 returned %d URLs, want 2", len(results))
		}
	})

	t.Run("list with offset", func(t *testing.T) {
		results, err := repo.List("", 2, 2)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(results) != 2 {
			t.Errorf("List() with offset=2 returned %d URLs, want 2", len(results))
		}

		// Should skip the first 2 (url4, url3) and return url2, url1
		if results[0].ShortCode != "url2" {
			t.Errorf("List() with offset=2 first URL = %v, want url2", results[0].ShortCode)
		}
	})
}

func TestSQLiteURLRepository_ListByCreatedByAndTimeRange(t *testing.T) {
	db, cleanup := setupSQLiteTestDB(t)
	defer cleanup()

	repo := NewSQLiteURLRepository(db)

	now := time.Now()
	
	// Create URLs at different times
	u1, _ := url.NewURL("time1", "https://example.com/1", "user1")
	u1.CreatedAt = now.Add(-2 * time.Hour)
	repo.Create(u1)

	u2, _ := url.NewURL("time2", "https://example.com/2", "user1")
	u2.CreatedAt = now.Add(-1 * time.Hour)
	repo.Create(u2)

	u3, _ := url.NewURL("time3", "https://example.com/3", "user2")
	u3.CreatedAt = now.Add(-30 * time.Minute)
	repo.Create(u3)

	t.Run("filter by user and time range", func(t *testing.T) {
		startTime := now.Add(-90 * time.Minute)
		endTime := now

		results, err := repo.ListByCreatedByAndTimeRange("user1", startTime, endTime)
		if err != nil {
			t.Fatalf("ListByCreatedByAndTimeRange() error = %v", err)
		}

		if len(results) != 1 {
			t.Errorf("ListByCreatedByAndTimeRange() returned %d URLs, want 1", len(results))
		}

		if len(results) > 0 && results[0].ShortCode != "time2" {
			t.Errorf("ListByCreatedByAndTimeRange() returned %v, want time2", results[0].ShortCode)
		}
	})

	t.Run("no URLs in time range", func(t *testing.T) {
		startTime := now.Add(1 * time.Hour)
		endTime := now.Add(2 * time.Hour)

		results, err := repo.ListByCreatedByAndTimeRange("user1", startTime, endTime)
		if err != nil {
			t.Fatalf("ListByCreatedByAndTimeRange() error = %v", err)
		}

		if len(results) != 0 {
			t.Errorf("ListByCreatedByAndTimeRange() returned %d URLs, want 0", len(results))
		}
	})
}

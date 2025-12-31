package repository

import (
	"context"
	"testing"
	"time"

	"github.com/matt-riley/mjrwtf/internal/domain/url"
	"github.com/matt-riley/mjrwtf/internal/domain/urlstatus"
)

func TestSQLiteURLStatusRepository_GetByURLID_NoRow(t *testing.T) {
	db, cleanup := setupSQLiteTestDB(t)
	defer cleanup()

	repo := NewSQLiteURLStatusRepository(db)

	status, err := repo.GetByURLID(context.Background(), 123)
	if err != nil {
		t.Fatalf("GetByURLID() error = %v", err)
	}
	if status != nil {
		t.Fatalf("GetByURLID() status = %+v, want nil", status)
	}
}

func TestSQLiteURLStatusRepository_Upsert_And_GetByURLID(t *testing.T) {
	db, cleanup := setupSQLiteTestDB(t)
	defer cleanup()

	urlRepo := NewSQLiteURLRepository(db)
	statusRepo := NewSQLiteURLStatusRepository(db)

	u, err := url.NewURL("s11", "https://example.com", "tester")
	if err != nil {
		t.Fatalf("NewURL() error = %v", err)
	}
	if err := urlRepo.Create(context.Background(), u); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	code := int64(200)
	archive := "https://web.archive.org/web/20200101000000/https://example.com"

	in := &urlstatus.URLStatus{
		URLID:            u.ID,
		LastCheckedAt:    &now,
		LastStatusCode:   &code,
		ArchiveURL:       &archive,
		ArchiveCheckedAt: &now,
	}
	if err := statusRepo.Upsert(context.Background(), in); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	out, err := statusRepo.GetByURLID(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("GetByURLID() error = %v", err)
	}
	if out == nil {
		t.Fatalf("GetByURLID() status = nil, want non-nil")
	}
	if out.URLID != u.ID {
		t.Fatalf("GetByURLID() URLID = %d, want %d", out.URLID, u.ID)
	}
	if out.LastCheckedAt == nil || !out.LastCheckedAt.Equal(now) {
		t.Fatalf("GetByURLID() LastCheckedAt = %v, want %v", out.LastCheckedAt, now)
	}
	if out.LastStatusCode == nil || *out.LastStatusCode != code {
		t.Fatalf("GetByURLID() LastStatusCode = %v, want %d", out.LastStatusCode, code)
	}
	if out.ArchiveURL == nil || *out.ArchiveURL != archive {
		t.Fatalf("GetByURLID() ArchiveURL = %v, want %q", out.ArchiveURL, archive)
	}
}

func TestSQLiteURLStatusRepository_ListDueForStatusCheck(t *testing.T) {
	db, cleanup := setupSQLiteTestDB(t)
	defer cleanup()

	urlRepo := NewSQLiteURLRepository(db)
	statusRepo := NewSQLiteURLStatusRepository(db)

	u, err := url.NewURL("due1", "https://example.com", "tester")
	if err != nil {
		t.Fatalf("NewURL() error = %v", err)
	}
	if err := urlRepo.Create(context.Background(), u); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	aliveCutoff := time.Now().Add(-1 * time.Hour)
	goneCutoff := time.Now().Add(-24 * time.Hour)

	due, err := statusRepo.ListDueForStatusCheck(context.Background(), aliveCutoff, goneCutoff, 10)
	if err != nil {
		t.Fatalf("ListDueForStatusCheck() error = %v", err)
	}
	if len(due) != 1 {
		t.Fatalf("ListDueForStatusCheck() len = %d, want %d", len(due), 1)
	}
	if due[0].URLID != u.ID {
		t.Fatalf("ListDueForStatusCheck() URLID = %d, want %d", due[0].URLID, u.ID)
	}
	if due[0].LastCheckedAt != nil {
		t.Fatalf("ListDueForStatusCheck() LastCheckedAt = %v, want nil", due[0].LastCheckedAt)
	}
}

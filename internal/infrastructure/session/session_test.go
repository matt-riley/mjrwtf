package session

import (
	"testing"
	"time"
)

func TestSession_Create(t *testing.T) {
	store := NewStore(24 * time.Hour)
	defer store.Shutdown()

	session, err := store.Create("user123")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if session.ID == "" {
		t.Error("expected non-empty session ID")
	}

	if session.UserID != "user123" {
		t.Errorf("expected UserID = user123, got %s", session.UserID)
	}

	if session.ExpiresAt.Before(time.Now()) {
		t.Error("expected session to not be expired")
	}
}

func TestSession_Get(t *testing.T) {
	store := NewStore(24 * time.Hour)
	defer store.Shutdown()

	created, err := store.Create("user456")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, exists := store.Get(created.ID)
	if !exists {
		t.Fatal("expected session to exist")
	}

	if retrieved.ID != created.ID {
		t.Errorf("expected ID = %s, got %s", created.ID, retrieved.ID)
	}

	if retrieved.UserID != "user456" {
		t.Errorf("expected UserID = user456, got %s", retrieved.UserID)
	}
}

func TestSession_Get_NotFound(t *testing.T) {
	store := NewStore(24 * time.Hour)
	defer store.Shutdown()

	_, exists := store.Get("nonexistent")
	if exists {
		t.Error("expected session to not exist")
	}
}

func TestSession_Get_Expired(t *testing.T) {
	store := NewStore(1 * time.Millisecond)
	defer store.Shutdown()

	created, err := store.Create("user789")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Wait for session to expire
	time.Sleep(10 * time.Millisecond)

	_, exists := store.Get(created.ID)
	if exists {
		t.Error("expected expired session to not be found")
	}
}

func TestSession_Delete(t *testing.T) {
	store := NewStore(24 * time.Hour)
	defer store.Shutdown()

	created, err := store.Create("user999")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify session exists
	_, exists := store.Get(created.ID)
	if !exists {
		t.Fatal("expected session to exist before deletion")
	}

	// Delete session
	store.Delete(created.ID)

	// Verify session no longer exists
	_, exists = store.Get(created.ID)
	if exists {
		t.Error("expected session to not exist after deletion")
	}
}

func TestSession_Refresh(t *testing.T) {
	store := NewStore(1 * time.Second)
	defer store.Shutdown()

	created, err := store.Create("user111")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	originalExpiry := created.ExpiresAt

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Refresh session
	err = store.Refresh(created.ID)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	// Get updated session
	refreshed, exists := store.Get(created.ID)
	if !exists {
		t.Fatal("expected session to exist after refresh")
	}

	if !refreshed.ExpiresAt.After(originalExpiry) {
		t.Error("expected expiry time to be extended after refresh")
	}
}

func TestGenerateSessionID(t *testing.T) {
	id1, err := generateSessionID()
	if err != nil {
		t.Fatalf("generateSessionID() error = %v", err)
	}

	id2, err := generateSessionID()
	if err != nil {
		t.Fatalf("generateSessionID() error = %v", err)
	}

	if id1 == id2 {
		t.Error("expected different session IDs to be generated")
	}

	if len(id1) == 0 {
		t.Error("expected non-empty session ID")
	}
}

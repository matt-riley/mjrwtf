package session

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

// Session represents a user session
type Session struct {
	ID        string
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// Store manages user sessions
type Store struct {
	sessions map[string]*Session
	mu       sync.RWMutex
	ttl      time.Duration
}

// NewStore creates a new session store with the given TTL
func NewStore(ttl time.Duration) *Store {
	store := &Store{
		sessions: make(map[string]*Session),
		ttl:      ttl,
	}
	
	// Start background cleanup goroutine
	go store.cleanup()
	
	return store
}

// Create creates a new session for the given user ID
func (s *Store) Create(userID string) (*Session, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, err
	}
	
	now := time.Now()
	session := &Session{
		ID:        sessionID,
		UserID:    userID,
		CreatedAt: now,
		ExpiresAt: now.Add(s.ttl),
	}
	
	s.mu.Lock()
	s.sessions[sessionID] = session
	s.mu.Unlock()
	
	return session, nil
}

// Get retrieves a session by ID
func (s *Store) Get(sessionID string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, false
	}
	
	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		return nil, false
	}
	
	return session, true
}

// Delete removes a session by ID
func (s *Store) Delete(sessionID string) {
	s.mu.Lock()
	delete(s.sessions, sessionID)
	s.mu.Unlock()
}

// Refresh extends the expiration time of a session
func (s *Store) Refresh(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	session, exists := s.sessions[sessionID]
	if !exists {
		return nil
	}
	
	session.ExpiresAt = time.Now().Add(s.ttl)
	return nil
}

// cleanup removes expired sessions periodically
func (s *Store) cleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for id, session := range s.sessions {
			if now.After(session.ExpiresAt) {
				delete(s.sessions, id)
			}
		}
		s.mu.Unlock()
	}
}

// generateSessionID generates a cryptographically secure random session ID
func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

package session

import (
	"strings"
	"testing"
	"time"
)

func TestNewSession(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		ipAddress string
		userAgent string
		wantErr   error
	}{
		{
			name:      "valid session",
			userID:    "authenticated-user",
			ipAddress: "192.168.1.1",
			userAgent: "Mozilla/5.0",
			wantErr:   nil,
		},
		{
			name:      "valid session with empty IP",
			userID:    "authenticated-user",
			ipAddress: "",
			userAgent: "Mozilla/5.0",
			wantErr:   nil,
		},
		{
			name:      "valid session with empty user agent",
			userID:    "authenticated-user",
			ipAddress: "192.168.1.1",
			userAgent: "",
			wantErr:   nil,
		},
		{
			name:      "valid session with long user agent",
			userID:    "authenticated-user",
			ipAddress: "192.168.1.1",
			userAgent: strings.Repeat("a", 500),
			wantErr:   nil,
		},
		{
			name:      "empty user ID",
			userID:    "",
			ipAddress: "192.168.1.1",
			userAgent: "Mozilla/5.0",
			wantErr:   ErrEmptyUserID,
		},
		{
			name:      "whitespace only user ID",
			userID:    "   ",
			ipAddress: "192.168.1.1",
			userAgent: "Mozilla/5.0",
			wantErr:   ErrEmptyUserID,
		},
		{
			name:      "user agent too long",
			userID:    "authenticated-user",
			ipAddress: "192.168.1.1",
			userAgent: strings.Repeat("a", 501),
			wantErr:   ErrUserAgentTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, err := NewSession(tt.userID, tt.ipAddress, tt.userAgent)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("NewSession() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if err != tt.wantErr && !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Errorf("NewSession() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("NewSession() unexpected error = %v", err)
				return
			}

			if len(session.ID) != SessionIDLength {
				t.Errorf("NewSession() ID length = %d, want %d", len(session.ID), SessionIDLength)
			}

			if !sessionIDRegex.MatchString(session.ID) {
				t.Errorf("NewSession() ID = %v, does not match hex pattern", session.ID)
			}

			if session.UserID != tt.userID {
				t.Errorf("NewSession() UserID = %v, want %v", session.UserID, tt.userID)
			}

			if session.IPAddress != tt.ipAddress {
				t.Errorf("NewSession() IPAddress = %v, want %v", session.IPAddress, tt.ipAddress)
			}

			if session.UserAgent != tt.userAgent {
				t.Errorf("NewSession() UserAgent = %v, want %v", session.UserAgent, tt.userAgent)
			}

			if session.CreatedAt.IsZero() {
				t.Error("NewSession() CreatedAt should not be zero")
			}

			if session.ExpiresAt.IsZero() {
				t.Error("NewSession() ExpiresAt should not be zero")
			}

			if session.LastActivityAt.IsZero() {
				t.Error("NewSession() LastActivityAt should not be zero")
			}

			expectedExpiresAt := session.CreatedAt.Add(DefaultSessionDuration)
			if !session.ExpiresAt.Equal(expectedExpiresAt) {
				t.Errorf("NewSession() ExpiresAt = %v, want %v", session.ExpiresAt, expectedExpiresAt)
			}

			if !session.LastActivityAt.Equal(session.CreatedAt) {
				t.Errorf("NewSession() LastActivityAt = %v, want %v", session.LastActivityAt, session.CreatedAt)
			}
		})
	}
}

func TestNewSession_UniqueIDs(t *testing.T) {
	// Test that multiple sessions get unique IDs
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		session, err := NewSession("user", "", "")
		if err != nil {
			t.Fatalf("NewSession() unexpected error = %v", err)
		}
		if ids[session.ID] {
			t.Errorf("NewSession() generated duplicate ID: %s", session.ID)
		}
		ids[session.ID] = true
	}
}

func TestValidateSessionID(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		wantErr   error
	}{
		{
			name:      "valid session ID",
			sessionID: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			wantErr:   nil,
		},
		{
			name:      "valid session ID with all zeros",
			sessionID: "0000000000000000000000000000000000000000000000000000000000000000",
			wantErr:   nil,
		},
		{
			name:      "valid session ID with all f",
			sessionID: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			wantErr:   nil,
		},
		{
			name:      "empty session ID",
			sessionID: "",
			wantErr:   ErrInvalidSessionID,
		},
		{
			name:      "too short",
			sessionID: "0123456789abcdef",
			wantErr:   ErrInvalidSessionID,
		},
		{
			name:      "too long",
			sessionID: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef00",
			wantErr:   ErrInvalidSessionID,
		},
		{
			name:      "uppercase hex",
			sessionID: "0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF",
			wantErr:   ErrInvalidSessionID,
		},
		{
			name:      "with special characters",
			sessionID: "0123456789abcdef-123456789abcdef0123456789abcdef0123456789abcdef",
			wantErr:   ErrInvalidSessionID,
		},
		{
			name:      "with spaces",
			sessionID: "0123456789abcdef 123456789abcdef0123456789abcdef0123456789abcdef",
			wantErr:   ErrInvalidSessionID,
		},
		{
			name:      "not hex characters",
			sessionID: "ghijklmnopqrstuvghijklmnopqrstuvghijklmnopqrstuvghijklmnopqrstuv",
			wantErr:   ErrInvalidSessionID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSessionID(tt.sessionID)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("ValidateSessionID() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateSessionID() unexpected error = %v", err)
			}
		})
	}
}

func TestSession_Validate(t *testing.T) {
	now := time.Now()
	validID := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	tests := []struct {
		name    string
		session *Session
		wantErr error
	}{
		{
			name: "valid session",
			session: &Session{
				ID:             validID,
				UserID:         "authenticated-user",
				CreatedAt:      now,
				ExpiresAt:      now.Add(24 * time.Hour),
				LastActivityAt: now,
				IPAddress:      "192.168.1.1",
				UserAgent:      "Mozilla/5.0",
			},
			wantErr: nil,
		},
		{
			name: "valid session with empty optional fields",
			session: &Session{
				ID:             validID,
				UserID:         "authenticated-user",
				CreatedAt:      now,
				ExpiresAt:      now.Add(24 * time.Hour),
				LastActivityAt: now,
				IPAddress:      "",
				UserAgent:      "",
			},
			wantErr: nil,
		},
		{
			name: "valid session with activity after creation",
			session: &Session{
				ID:             validID,
				UserID:         "authenticated-user",
				CreatedAt:      now,
				ExpiresAt:      now.Add(24 * time.Hour),
				LastActivityAt: now.Add(1 * time.Hour),
				IPAddress:      "192.168.1.1",
				UserAgent:      "Mozilla/5.0",
			},
			wantErr: nil,
		},
		{
			name: "invalid session ID",
			session: &Session{
				ID:             "invalid",
				UserID:         "authenticated-user",
				CreatedAt:      now,
				ExpiresAt:      now.Add(24 * time.Hour),
				LastActivityAt: now,
			},
			wantErr: ErrInvalidSessionID,
		},
		{
			name: "empty user ID",
			session: &Session{
				ID:             validID,
				UserID:         "",
				CreatedAt:      now,
				ExpiresAt:      now.Add(24 * time.Hour),
				LastActivityAt: now,
			},
			wantErr: ErrEmptyUserID,
		},
		{
			name: "whitespace only user ID",
			session: &Session{
				ID:             validID,
				UserID:         "   ",
				CreatedAt:      now,
				ExpiresAt:      now.Add(24 * time.Hour),
				LastActivityAt: now,
			},
			wantErr: ErrEmptyUserID,
		},
		{
			name: "expires at before created at",
			session: &Session{
				ID:             validID,
				UserID:         "authenticated-user",
				CreatedAt:      now,
				ExpiresAt:      now.Add(-1 * time.Hour),
				LastActivityAt: now,
			},
			wantErr: ErrInvalidTimestamps,
		},
		{
			name: "expires at equal to created at",
			session: &Session{
				ID:             validID,
				UserID:         "authenticated-user",
				CreatedAt:      now,
				ExpiresAt:      now,
				LastActivityAt: now,
			},
			wantErr: ErrInvalidTimestamps,
		},
		{
			name: "last activity before created at",
			session: &Session{
				ID:             validID,
				UserID:         "authenticated-user",
				CreatedAt:      now,
				ExpiresAt:      now.Add(24 * time.Hour),
				LastActivityAt: now.Add(-1 * time.Hour),
			},
			wantErr: ErrInvalidTimestamps,
		},
		{
			name: "user agent too long",
			session: &Session{
				ID:             validID,
				UserID:         "authenticated-user",
				CreatedAt:      now,
				ExpiresAt:      now.Add(24 * time.Hour),
				LastActivityAt: now,
				UserAgent:      strings.Repeat("a", 501),
			},
			wantErr: ErrUserAgentTooLong,
		},
		{
			name: "user agent exactly max length",
			session: &Session{
				ID:             validID,
				UserID:         "authenticated-user",
				CreatedAt:      now,
				ExpiresAt:      now.Add(24 * time.Hour),
				LastActivityAt: now,
				UserAgent:      strings.Repeat("a", 500),
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.session.Validate()
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Session.Validate() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if err != tt.wantErr && !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Errorf("Session.Validate() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("Session.Validate() unexpected error = %v", err)
			}
		})
	}
}

func TestSession_IsExpired(t *testing.T) {
	now := time.Now()
	validID := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	tests := []struct {
		name        string
		session     *Session
		wantExpired bool
	}{
		{
			name: "not expired - future expiry",
			session: &Session{
				ID:        validID,
				UserID:    "user",
				CreatedAt: now,
				ExpiresAt: now.Add(1 * time.Hour),
			},
			wantExpired: false,
		},
		{
			name: "expired - past expiry",
			session: &Session{
				ID:        validID,
				UserID:    "user",
				CreatedAt: now.Add(-2 * time.Hour),
				ExpiresAt: now.Add(-1 * time.Hour),
			},
			wantExpired: true,
		},
		{
			name: "not expired - just created",
			session: &Session{
				ID:        validID,
				UserID:    "user",
				CreatedAt: now,
				ExpiresAt: now.Add(24 * time.Hour),
			},
			wantExpired: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.session.IsExpired(); got != tt.wantExpired {
				t.Errorf("Session.IsExpired() = %v, want %v", got, tt.wantExpired)
			}
		})
	}
}

func TestSession_IsIdle(t *testing.T) {
	now := time.Now()
	validID := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	tests := []struct {
		name        string
		session     *Session
		idleTimeout time.Duration
		wantIdle    bool
	}{
		{
			name: "not idle - recent activity",
			session: &Session{
				ID:             validID,
				UserID:         "user",
				CreatedAt:      now.Add(-1 * time.Hour),
				LastActivityAt: now.Add(-5 * time.Minute),
			},
			idleTimeout: 10 * time.Minute,
			wantIdle:    false,
		},
		{
			name: "idle - old activity",
			session: &Session{
				ID:             validID,
				UserID:         "user",
				CreatedAt:      now.Add(-2 * time.Hour),
				LastActivityAt: now.Add(-20 * time.Minute),
			},
			idleTimeout: 10 * time.Minute,
			wantIdle:    true,
		},
		{
			name: "not idle - just at threshold",
			session: &Session{
				ID:             validID,
				UserID:         "user",
				CreatedAt:      now.Add(-1 * time.Hour),
				LastActivityAt: now.Add(-10 * time.Minute),
			},
			idleTimeout: 10*time.Minute + time.Second,
			wantIdle:    false,
		},
		{
			name: "idle - exactly at threshold",
			session: &Session{
				ID:             validID,
				UserID:         "user",
				CreatedAt:      now.Add(-1 * time.Hour),
				LastActivityAt: now.Add(-10 * time.Minute),
			},
			idleTimeout: 10 * time.Minute,
			wantIdle:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.session.IsIdle(tt.idleTimeout); got != tt.wantIdle {
				t.Errorf("Session.IsIdle() = %v, want %v", got, tt.wantIdle)
			}
		})
	}
}

func TestSession_UpdateActivity(t *testing.T) {
	now := time.Now()
	validID := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	session := &Session{
		ID:             validID,
		UserID:         "user",
		CreatedAt:      now.Add(-1 * time.Hour),
		ExpiresAt:      now.Add(23 * time.Hour),
		LastActivityAt: now.Add(-30 * time.Minute),
	}

	oldActivity := session.LastActivityAt

	// Sleep a tiny bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	session.UpdateActivity()

	if !session.LastActivityAt.After(oldActivity) {
		t.Errorf("Session.UpdateActivity() LastActivityAt = %v, should be after %v", session.LastActivityAt, oldActivity)
	}

	if session.LastActivityAt.Before(now) {
		t.Errorf("Session.UpdateActivity() LastActivityAt = %v, should be >= %v", session.LastActivityAt, now)
	}
}

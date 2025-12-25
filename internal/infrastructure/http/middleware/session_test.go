package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/matt-riley/mjrwtf/internal/infrastructure/session"
)

func TestSessionMiddleware_ValidSession(t *testing.T) {
	store := session.NewStore(24 * time.Hour)
	
	// Create a session
	sess, err := store.Create("test-user")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	
	// Create middleware
	middleware := SessionMiddleware(store)
	
	// Create test handler that checks for user ID in context
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetSessionUserID(r.Context())
		if !ok {
			t.Error("expected user ID in context")
		}
		if userID != "test-user" {
			t.Errorf("expected userID = test-user, got %s", userID)
		}
		w.WriteHeader(http.StatusOK)
	}))
	
	// Create request with session cookie
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  SessionCookieName,
		Value: sess.ID,
	})
	
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

func TestSessionMiddleware_NoSession(t *testing.T) {
	store := session.NewStore(24 * time.Hour)
	
	// Create middleware
	middleware := SessionMiddleware(store)
	
	// Create test handler that checks for user ID in context
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := GetSessionUserID(r.Context())
		if ok {
			t.Error("expected no user ID in context")
		}
		w.WriteHeader(http.StatusOK)
	}))
	
	// Create request without session cookie
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

func TestSessionMiddleware_ExpiredSession(t *testing.T) {
	store := session.NewStore(1 * time.Millisecond)
	
	// Create a session
	sess, err := store.Create("test-user")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	
	// Wait for session to expire
	time.Sleep(10 * time.Millisecond)
	
	// Create middleware
	middleware := SessionMiddleware(store)
	
	// Create test handler that checks for user ID in context
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := GetSessionUserID(r.Context())
		if ok {
			t.Error("expected no user ID in context for expired session")
		}
		w.WriteHeader(http.StatusOK)
	}))
	
	// Create request with expired session cookie
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  SessionCookieName,
		Value: sess.ID,
	})
	
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

func TestRequireSession_ValidSession(t *testing.T) {
	store := session.NewStore(24 * time.Hour)
	
	// Create a session
	sess, err := store.Create("test-user")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	
	// Create middleware chain
	sessionMiddleware := SessionMiddleware(store)
	requireMiddleware := RequireSession(store, "/login")
	
	// Create test handler
	handler := sessionMiddleware(requireMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("protected content"))
	})))
	
	// Create request with session cookie
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  SessionCookieName,
		Value: sess.ID,
	})
	
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
	
	if rr.Body.String() != "protected content" {
		t.Errorf("expected protected content, got %s", rr.Body.String())
	}
}

func TestRequireSession_NoSession(t *testing.T) {
	store := session.NewStore(24 * time.Hour)
	
	// Create middleware chain
	sessionMiddleware := SessionMiddleware(store)
	requireMiddleware := RequireSession(store, "/login")
	
	// Create test handler
	handler := sessionMiddleware(requireMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("protected content"))
	})))
	
	// Create request without session cookie
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	
	if rr.Code != http.StatusSeeOther {
		t.Errorf("expected status 303, got %d", rr.Code)
	}
	
	location := rr.Header().Get("Location")
	if location != "/login" {
		t.Errorf("expected redirect to /login, got %s", location)
	}
}

func TestSetSessionCookie(t *testing.T) {
	tests := []struct {
		name   string
		secure bool
	}{
		{"with secure flag", true},
		{"without secure flag", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			SetSessionCookie(rr, "test-session-id", 3600, tt.secure)

			cookies := rr.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatalf("expected 1 cookie, got %d", len(cookies))
			}

			cookie := cookies[0]
			if cookie.Name != SessionCookieName {
				t.Errorf("expected cookie name %s, got %s", SessionCookieName, cookie.Name)
			}

			if cookie.Value != "test-session-id" {
				t.Errorf("expected cookie value test-session-id, got %s", cookie.Value)
			}

			if !cookie.HttpOnly {
				t.Error("expected HttpOnly cookie")
			}

			if cookie.Secure != tt.secure {
				t.Errorf("expected Secure %v, got %v", tt.secure, cookie.Secure)
			}

			if cookie.MaxAge != 3600 {
				t.Errorf("expected MaxAge 3600, got %d", cookie.MaxAge)
			}
		})
	}
}

func TestClearSessionCookie(t *testing.T) {
	tests := []struct {
		name   string
		secure bool
	}{
		{"with secure flag", true},
		{"without secure flag", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			ClearSessionCookie(rr, tt.secure)

			cookies := rr.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatalf("expected 1 cookie, got %d", len(cookies))
			}

			cookie := cookies[0]
			if cookie.Name != SessionCookieName {
				t.Errorf("expected cookie name %s, got %s", SessionCookieName, cookie.Name)
			}

			if cookie.Value != "" {
				t.Errorf("expected empty cookie value, got %s", cookie.Value)
			}

			if cookie.Secure != tt.secure {
				t.Errorf("expected Secure %v, got %v", tt.secure, cookie.Secure)
			}

			if cookie.MaxAge != -1 {
				t.Errorf("expected MaxAge -1, got %d", cookie.MaxAge)
			}
		})
	}
}

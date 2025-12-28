package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSessionOrBearerAuth_SessionWins(t *testing.T) {
	mw := SessionOrBearerAuth([]string{"test-token"})

	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserID(r.Context())
		if !ok {
			t.Fatalf("expected user id in context")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(userID))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), SessionUserIDKey, "session-user"))
	req.Header.Set("Authorization", "Bearer wrong-token")

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if got := w.Body.String(); got != "session-user" {
		t.Fatalf("expected session user, got %q", got)
	}
}

func TestSessionOrBearerAuth_BearerUsedWhenNoSession(t *testing.T) {
	mw := SessionOrBearerAuth([]string{"test-token"})

	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserID(r.Context())
		if !ok {
			t.Fatalf("expected user id in context")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(userID))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if got := w.Body.String(); got != "authenticated-user" {
		t.Fatalf("expected bearer user, got %q", got)
	}
}

func TestSessionOrBearerAuth_UnauthorizedWhenNoSessionOrBearer(t *testing.T) {
	mw := SessionOrBearerAuth([]string{"test-token"})

	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

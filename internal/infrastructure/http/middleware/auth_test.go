package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuth_MissingToken(t *testing.T) {
	authMiddleware := Auth([]string{"test-secret-token"})
	handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("protected resource"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	expected := `{"error":"Unauthorized: missing authorization header"}
`
	if rec.Body.String() != expected {
		t.Errorf("expected response %q, got %q", expected, rec.Body.String())
	}
}

func TestAuth_InvalidTokenFormat(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
	}{
		{"no bearer prefix", "test-token"},
		{"wrong prefix", "Basic test-token"},
		{"empty token", "Bearer "},
		{"missing token part", "Bearer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authMiddleware := Auth([]string{"test-secret-token"})
			handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", tt.authHeader)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
			}
		})
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	authMiddleware := Auth([]string{"correct-secret-token"})
	handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("protected resource"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	expected := `{"error":"Unauthorized: invalid token"}
`
	if rec.Body.String() != expected {
		t.Errorf("expected response %q, got %q", expected, rec.Body.String())
	}
}

func TestAuth_ValidToken(t *testing.T) {
	authMiddleware := Auth([]string{"test-secret-token"})
	handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("protected resource"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer test-secret-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	expected := "protected resource"
	if rec.Body.String() != expected {
		t.Errorf("expected response %q, got %q", expected, rec.Body.String())
	}
}

func TestAuth_ValidToken_MultipleTokens(t *testing.T) {
	authMiddleware := Auth([]string{"old-token", "new-token"})
	handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tokens := []string{"old-token", "new-token"}
	for _, token := range tokens {
		t.Run(token, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
			}
		})
	}
}

func TestAuth_UserIdentityInContext(t *testing.T) {
	authMiddleware := Auth([]string{"test-secret-token"})

	var capturedUserID string
	var userIDFound bool

	handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID, userIDFound = GetUserID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer test-secret-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if !userIDFound {
		t.Error("expected user ID to be present in context")
	}

	if capturedUserID != "authenticated-user" {
		t.Errorf("expected user ID %q, got %q", "authenticated-user", capturedUserID)
	}
}

func TestAuth_MultipleRequests(t *testing.T) {
	authMiddleware := Auth([]string{"test-secret-token"})
	handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{"valid token", "Bearer test-secret-token", http.StatusOK},
		{"invalid token", "Bearer wrong-token", http.StatusUnauthorized},
		{"no token", "", http.StatusUnauthorized},
		{"valid token again", "Bearer test-secret-token", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestGetUserID_NoContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	userID, ok := GetUserID(req.Context())

	if ok {
		t.Error("expected ok to be false when no user ID in context")
	}

	if userID != "" {
		t.Errorf("expected empty user ID, got %q", userID)
	}
}

func TestAuth_CaseSensitiveBearer(t *testing.T) {
	authMiddleware := Auth([]string{"test-secret-token"})
	handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{"correct case Bearer", "Bearer test-secret-token", http.StatusOK},
		{"lowercase bearer", "bearer test-secret-token", http.StatusUnauthorized},
		{"uppercase BEARER", "BEARER test-secret-token", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", tt.authHeader)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

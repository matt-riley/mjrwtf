package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"

	"github.com/matt-riley/mjrwtf/internal/adapters/repository"
	"github.com/matt-riley/mjrwtf/internal/application"
	"github.com/matt-riley/mjrwtf/internal/domain/click"
	"github.com/matt-riley/mjrwtf/internal/domain/url"
	"github.com/matt-riley/mjrwtf/internal/migrations"
)

func TestAnalyticsIntegration(t *testing.T) {
	// Setup test database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Apply migrations
	goose.SetBaseFS(migrations.SQLiteMigrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("failed to set dialect: %v", err)
	}
	if err := goose.Up(db, "sqlite"); err != nil {
		t.Fatalf("failed to apply migrations: %v", err)
	}

	// Setup server
	cfg := testConfig()

	server, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer server.Shutdown(context.Background())

	// Create test data
	urlRepo := repository.NewSQLiteURLRepository(db)
	clickRepo := repository.NewSQLiteClickRepository(db)

	testURL, err := url.NewURL("test123", "https://example.com", "authenticated-user")
	if err != nil {
		t.Fatalf("failed to create test URL: %v", err)
	}

	if err := urlRepo.Create(context.Background(), testURL); err != nil {
		t.Fatalf("failed to save test URL: %v", err)
	}

	// Record some clicks
	ctx := context.Background()
	clicks := []struct {
		referrer  string
		country   string
		userAgent string
	}{
		{"https://google.com", "US", "Mozilla/5.0"},
		{"https://google.com", "US", "Chrome/90.0"},
		{"https://twitter.com", "UK", "Safari/14.0"},
		{"", "CA", "Firefox/88.0"},
		{"https://google.com", "US", "Edge/90.0"},
	}

	for _, c := range clicks {
		clickEntity, err := click.NewClick(testURL.ID, c.referrer, c.country, c.userAgent)
		if err != nil {
			t.Fatalf("failed to create click: %v", err)
		}
		if err := clickRepo.Record(ctx, clickEntity); err != nil {
			t.Fatalf("failed to record click: %v", err)
		}
	}

	t.Run("get analytics without authentication", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/urls/test123/analytics", nil)
		w := httptest.NewRecorder()

		server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})

	t.Run("get analytics with authentication", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/urls/test123/analytics", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp application.GetAnalyticsResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.ShortCode != "test123" {
			t.Errorf("expected short_code test123, got %s", resp.ShortCode)
		}

		if resp.OriginalURL != "https://example.com" {
			t.Errorf("expected original_url https://example.com, got %s", resp.OriginalURL)
		}

		if resp.TotalClicks != 5 {
			t.Errorf("expected 5 total clicks, got %d", resp.TotalClicks)
		}

		if resp.ByCountry["US"] != 3 {
			t.Errorf("expected 3 US clicks, got %d", resp.ByCountry["US"])
		}

		if resp.ByCountry["UK"] != 1 {
			t.Errorf("expected 1 UK click, got %d", resp.ByCountry["UK"])
		}

		if resp.ByCountry["CA"] != 1 {
			t.Errorf("expected 1 CA click, got %d", resp.ByCountry["CA"])
		}

		if resp.ByReferrer["https://google.com"] != 3 {
			t.Errorf("expected 3 google.com clicks, got %d", resp.ByReferrer["https://google.com"])
		}

		if resp.ByReferrer["https://twitter.com"] != 1 {
			t.Errorf("expected 1 twitter.com click, got %d", resp.ByReferrer["https://twitter.com"])
		}

		// ByDate should be present for all-time stats
		if resp.ByDate == nil {
			t.Error("expected by_date to be present")
		}

		// Time range should be nil for all-time stats
		if resp.StartTime != nil || resp.EndTime != nil {
			t.Error("expected start_time and end_time to be nil for all-time stats")
		}
	})

	t.Run("get analytics with time range", func(t *testing.T) {
		now := time.Now().UTC()
		startTime := now.Add(-24 * time.Hour).Format(time.RFC3339)
		endTime := now.Add(24 * time.Hour).Format(time.RFC3339)

		url := fmt.Sprintf("/api/urls/test123/analytics?start_time=%s&end_time=%s", startTime, endTime)
		req := httptest.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp application.GetAnalyticsResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.TotalClicks != 5 {
			t.Errorf("expected 5 total clicks, got %d", resp.TotalClicks)
		}

		// ByDate should not be present for time range stats
		if resp.ByDate != nil {
			t.Error("expected by_date to be nil for time range stats")
		}

		// Time range should be present
		if resp.StartTime == nil || resp.EndTime == nil {
			t.Error("expected start_time and end_time to be present for time range stats")
		}
	})

	t.Run("get analytics for non-existent URL", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/urls/notfound/analytics", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("unauthorized access to analytics (URL owned by different user)", func(t *testing.T) {
		// Create a URL with a different owner
		otherUserURL, err := url.NewURL("other123", "https://other.com", "other-user")
		if err != nil {
			t.Fatalf("failed to create test URL: %v", err)
		}

		if err := urlRepo.Create(context.Background(), otherUserURL); err != nil {
			t.Fatalf("failed to save test URL: %v", err)
		}

		// Try to access analytics for URL owned by other-user
		req := httptest.NewRequest(http.MethodGet, "/api/urls/other123/analytics", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		server.Router().ServeHTTP(w, req)

		// Should get 403 Forbidden because authenticated-user doesn't own this URL
		if w.Code != http.StatusForbidden {
			t.Errorf("expected status 403, got %d: %s", w.Code, w.Body.String())
		}
	})
}

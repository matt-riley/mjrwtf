package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_CreateURL_BuildsRequestAndDecodesResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected method POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/urls" {
			t.Fatalf("expected path /api/urls, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("expected Authorization header, got %q", got)
		}

		var req CreateURLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.OriginalURL != "https://example.com" {
			t.Fatalf("expected original_url, got %q", req.OriginalURL)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"short_code":"abc123","short_url":"http://localhost:8080/abc123","original_url":"https://example.com"}`))
	}))
	defer ts.Close()

	c, err := New(ts.URL, WithToken("test-token"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	resp, err := c.CreateURL(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("CreateURL: %v", err)
	}
	if resp.ShortCode != "abc123" {
		t.Fatalf("expected short_code abc123, got %q", resp.ShortCode)
	}
}

func TestClient_ListURLs_AddsQueryParams(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected method GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/urls" {
			t.Fatalf("expected path /api/urls, got %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("limit"); got != "20" {
			t.Fatalf("expected limit=20, got %q", got)
		}
		if got := r.URL.Query().Get("offset"); got != "40" {
			t.Fatalf("expected offset=40, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"urls":[],"total":0,"limit":20,"offset":40}`))
	}))
	defer ts.Close()

	c, err := New(ts.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := c.ListURLs(context.Background(), 20, 40); err != nil {
		t.Fatalf("ListURLs: %v", err)
	}
}

func TestClient_DecodesErrorResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer ts.Close()

	c, err := New(ts.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = c.ListURLs(context.Background(), 0, 0)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", apiErr.StatusCode)
	}
	if apiErr.Message != "unauthorized" {
		t.Fatalf("expected message unauthorized, got %q", apiErr.Message)
	}
}

func TestClient_RateLimit_SurfacesRetryAfter(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"Too Many Requests"}`))
	}))
	defer ts.Close()

	c, err := New(ts.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = c.ListURLs(context.Background(), 0, 0)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.RetryAfter != 5*time.Second {
		t.Fatalf("expected RetryAfter 5s, got %s", apiErr.RetryAfter)
	}
}

func TestClient_DeleteURL_BuildsRequest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("expected method DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/urls/abc123" {
			t.Fatalf("expected path /api/urls/abc123, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	c, err := New(ts.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := c.DeleteURL(context.Background(), "abc123"); err != nil {
		t.Fatalf("DeleteURL: %v", err)
	}
}

func TestClient_GetAnalytics_BuildsRequestAndDecodesResponse(t *testing.T) {
	start := time.Date(2025, 11, 20, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 11, 22, 23, 59, 59, 0, time.UTC)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected method GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/urls/abc123/analytics" {
			t.Fatalf("expected path /api/urls/abc123/analytics, got %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("start_time"); got != start.Format(time.RFC3339) {
			t.Fatalf("expected start_time %q, got %q", start.Format(time.RFC3339), got)
		}
		if got := r.URL.Query().Get("end_time"); got != end.Format(time.RFC3339) {
			t.Fatalf("expected end_time %q, got %q", end.Format(time.RFC3339), got)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"short_code":"abc123","original_url":"https://example.com","total_clicks":150,"by_country":{"US":75},"by_referrer":{"direct":60}}`))
	}))
	defer ts.Close()

	c, err := New(ts.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	resp, err := c.GetAnalytics(context.Background(), "abc123", &start, &end)
	if err != nil {
		t.Fatalf("GetAnalytics: %v", err)
	}
	if resp.ShortCode != "abc123" {
		t.Fatalf("expected short_code abc123, got %q", resp.ShortCode)
	}
	if resp.TotalClicks != 150 {
		t.Fatalf("expected total_clicks 150, got %d", resp.TotalClicks)
	}
}

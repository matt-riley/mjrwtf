package tui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/matt-riley/mjrwtf/internal/tui/tui_config"
)

func TestGetAnalyticsCmd(t *testing.T) {
	start := time.Date(2025, 11, 20, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 11, 22, 23, 59, 59, 0, time.UTC)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/api/urls/abc123/analytics" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("start_time") != start.Format(time.RFC3339) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.URL.Query().Get("end_time") != end.Format(time.RFC3339) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"short_code":"abc123","original_url":"https://example.com","total_clicks":3,"by_country":{"US":2},"by_referrer":{"direct":3},"start_time":"2025-11-20T00:00:00Z","end_time":"2025-11-22T23:59:59Z"}`)
	}))
	t.Cleanup(srv.Close)

	msg := getAnalyticsCmd(tui_config.Config{BaseURL: srv.URL, Token: "t"}, "abc123", &start, &end)().(getAnalyticsMsg)
	if msg.err != nil {
		t.Fatalf("expected nil err, got %v", msg.err)
	}
	if msg.resp == nil {
		t.Fatalf("expected resp")
	}
	if msg.resp.ShortCode != "abc123" {
		t.Fatalf("ShortCode=%q", msg.resp.ShortCode)
	}
	if msg.resp.TotalClicks != 3 {
		t.Fatalf("TotalClicks=%d", msg.resp.TotalClicks)
	}
}

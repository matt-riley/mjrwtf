package metrics

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNew_CreatesMetrics(t *testing.T) {
	m := New()

	if m.Registry == nil {
		t.Error("expected registry to be initialized")
	}
	if m.HTTPRequestsTotal == nil {
		t.Error("expected HTTPRequestsTotal to be initialized")
	}
	if m.HTTPRequestDuration == nil {
		t.Error("expected HTTPRequestDuration to be initialized")
	}
	if m.URLClicksTotal == nil {
		t.Error("expected URLClicksTotal to be initialized")
	}
	if m.URLsActiveTotal == nil {
		t.Error("expected URLsActiveTotal to be initialized")
	}
	if m.RedirectClickQueueDepth == nil {
		t.Error("expected RedirectClickQueueDepth to be initialized")
	}
	if m.RedirectClickDroppedTotal == nil {
		t.Error("expected RedirectClickDroppedTotal to be initialized")
	}
	if m.RedirectClickRecordFailuresTotal == nil {
		t.Error("expected RedirectClickRecordFailuresTotal to be initialized")
	}
}

func TestMetrics_RecordHTTPRequest(t *testing.T) {
	m := New()

	m.RecordHTTPRequest("GET", "/test", "200", 0.5)

	count := testutil.ToFloat64(m.HTTPRequestsTotal.WithLabelValues("GET", "/test", "200"))
	if count != 1 {
		t.Errorf("expected 1 request, got %f", count)
	}
}

func TestMetrics_RecordURLClick(t *testing.T) {
	m := New()

	m.RecordURLClick("abc123")
	m.RecordURLClick("abc123")
	m.RecordURLClick("xyz789")

	count := testutil.ToFloat64(m.URLClicksTotal.WithLabelValues("all"))
	if count != 3 {
		t.Errorf("expected 3 total clicks, got %f", count)
	}
}

func TestMetrics_SetActiveURLs(t *testing.T) {
	m := New()

	m.SetActiveURLs(42)

	count := testutil.ToFloat64(m.URLsActiveTotal)
	if count != 42 {
		t.Errorf("expected 42 active URLs, got %f", count)
	}
}

func TestMetrics_IncrementDecrementActiveURLs(t *testing.T) {
	m := New()

	m.SetActiveURLs(10)
	m.IncrementActiveURLs()
	m.IncrementActiveURLs()
	m.DecrementActiveURLs()

	count := testutil.ToFloat64(m.URLsActiveTotal)
	if count != 11 {
		t.Errorf("expected 11 active URLs, got %f", count)
	}
}

func TestMetrics_Handler(t *testing.T) {
	m := New()

	// Record some metrics first
	m.RecordHTTPRequest("GET", "/test", "200", 0.1)
	m.RecordURLClick("abc123")
	m.SetActiveURLs(5)

	handler := m.Handler()

	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()

	// Check for expected metrics in output
	if !strings.Contains(body, "mjrwtf_http_requests_total") {
		t.Error("expected http_requests_total metric in output")
	}
	if !strings.Contains(body, "mjrwtf_http_request_duration_seconds") {
		t.Error("expected http_request_duration_seconds metric in output")
	}
	if !strings.Contains(body, "mjrwtf_url_clicks_total") {
		t.Error("expected url_clicks_total metric in output")
	}
	if !strings.Contains(body, "mjrwtf_urls_active_total") {
		t.Error("expected urls_active_total metric in output")
	}
	// Check for Go runtime metrics
	if !strings.Contains(body, "go_goroutines") {
		t.Error("expected go_goroutines metric in output")
	}
}

func TestNew_MultipleInstances(t *testing.T) {
	// Test that multiple instances can be created without panicking
	// (important for tests that create multiple servers)
	m1 := New()
	m2 := New()

	if m1.Registry == m2.Registry {
		t.Error("expected separate registries for each instance")
	}

	// Both should work independently
	m1.RecordHTTPRequest("GET", "/test1", "200", 0.1)
	m2.RecordHTTPRequest("POST", "/test2", "201", 0.2)

	count1 := testutil.ToFloat64(m1.HTTPRequestsTotal.WithLabelValues("GET", "/test1", "200"))
	if count1 != 1 {
		t.Errorf("expected 1 request in m1, got %f", count1)
	}

	count2 := testutil.ToFloat64(m2.HTTPRequestsTotal.WithLabelValues("POST", "/test2", "201"))
	if count2 != 1 {
		t.Errorf("expected 1 request in m2, got %f", count2)
	}
}

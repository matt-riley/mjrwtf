package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

// mockHTTPClient is a mock HTTP client for testing
type mockHTTPClient struct {
	doFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.doFunc(req)
}

func TestNewDiscordNotifier(t *testing.T) {
	webhookURL := "https://discord.com/api/webhooks/test"
	notifier := NewDiscordNotifier(webhookURL)

	if notifier.webhookURL != webhookURL {
		t.Errorf("expected webhook URL %s, got %s", webhookURL, notifier.webhookURL)
	}

	if notifier.client == nil {
		t.Error("expected HTTP client to be initialized")
	}

	if notifier.rateLimiter == nil {
		t.Error("expected rate limiter to be initialized")
	}

	if !notifier.sendAsync {
		t.Error("expected async sending to be enabled by default")
	}
}

func TestDiscordNotifier_IsEnabled(t *testing.T) {
	tests := []struct {
		name       string
		webhookURL string
		want       bool
	}{
		{
			name:       "enabled with webhook URL",
			webhookURL: "https://discord.com/api/webhooks/test",
			want:       true,
		},
		{
			name:       "disabled without webhook URL",
			webhookURL: "",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifier := NewDiscordNotifier(tt.webhookURL)
			if got := notifier.IsEnabled(); got != tt.want {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDiscordNotifier_NotifyError_SendsSuccessfully(t *testing.T) {
	var capturedRequest *http.Request
	var capturedBody []byte

	mockClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			capturedRequest = req
			capturedBody, _ = io.ReadAll(req.Body)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString("")),
			}, nil
		}),
	}

	notifier := NewDiscordNotifier(
		"https://discord.com/api/webhooks/test",
		WithHTTPClient(mockClient),
		WithAsyncSend(false), // Synchronous for testing
	)

	ctx := context.Background()
	errCtx := ErrorContext{
		ErrorMessage: "test error",
		StackTrace:   "stack trace here",
		RequestID:    "req-123",
		Method:       "GET",
		Path:         "/test",
		UserID:       "user-456",
		Timestamp:    time.Now(),
	}

	notifier.NotifyError(ctx, errCtx)

	if capturedRequest == nil {
		t.Fatal("expected HTTP request to be made")
	}

	if capturedRequest.Method != "POST" {
		t.Errorf("expected POST method, got %s", capturedRequest.Method)
	}

	if capturedRequest.Header.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", capturedRequest.Header.Get("Content-Type"))
	}

	// Verify payload structure
	var payload map[string]interface{}
	if err := json.Unmarshal(capturedBody, &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	embeds, ok := payload["embeds"].([]interface{})
	if !ok || len(embeds) == 0 {
		t.Fatal("expected embeds in payload")
	}

	embed := embeds[0].(map[string]interface{})
	if embed["title"] != "🚨 Critical Error Detected" {
		t.Errorf("unexpected embed title: %v", embed["title"])
	}
}

func TestDiscordNotifier_NotifyError_RateLimiting(t *testing.T) {
	requestCount := 0

	mockClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requestCount++
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString("")),
			}, nil
		}),
	}

	notifier := NewDiscordNotifier(
		"https://discord.com/api/webhooks/test",
		WithHTTPClient(mockClient),
		WithAsyncSend(false),
		WithRateLimit(500*time.Millisecond),
	)

	ctx := context.Background()
	errCtx := ErrorContext{
		ErrorMessage: "same error message",
		Timestamp:    time.Now(),
	}

	// Send first notification - should succeed
	notifier.NotifyError(ctx, errCtx)

	// Send second notification immediately - should be rate limited
	notifier.NotifyError(ctx, errCtx)

	if requestCount != 1 {
		t.Errorf("expected 1 request due to rate limiting, got %d", requestCount)
	}

	// Wait for rate limit to expire
	time.Sleep(600 * time.Millisecond)

	// Send third notification - should succeed
	notifier.NotifyError(ctx, errCtx)

	if requestCount != 2 {
		t.Errorf("expected 2 requests after rate limit expires, got %d", requestCount)
	}
}

func TestDiscordNotifier_NotifyError_DifferentErrorTypes(t *testing.T) {
	requestCount := 0

	mockClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requestCount++
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString("")),
			}, nil
		}),
	}

	notifier := NewDiscordNotifier(
		"https://discord.com/api/webhooks/test",
		WithHTTPClient(mockClient),
		WithAsyncSend(false),
	)

	ctx := context.Background()

	// Send different error types - each should go through
	notifier.NotifyError(ctx, ErrorContext{
		ErrorMessage: "error type 1",
		Timestamp:    time.Now(),
	})

	notifier.NotifyError(ctx, ErrorContext{
		ErrorMessage: "error type 2",
		Timestamp:    time.Now(),
	})

	if requestCount != 2 {
		t.Errorf("expected 2 requests for different error types, got %d", requestCount)
	}
}

func TestDiscordNotifier_NotifyError_HandlesDiscordAPIFailure(t *testing.T) {
	mockClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 500,
				Body:       io.NopCloser(bytes.NewBufferString("Internal Server Error")),
			}, nil
		}),
	}

	notifier := NewDiscordNotifier(
		"https://discord.com/api/webhooks/test",
		WithHTTPClient(mockClient),
		WithAsyncSend(false),
	)

	ctx := context.Background()
	errCtx := ErrorContext{
		ErrorMessage: "test error",
		Timestamp:    time.Now(),
	}

	// Should not panic or crash
	notifier.NotifyError(ctx, errCtx)
}

func TestDiscordNotifier_NotifyError_HandlesNetworkError(t *testing.T) {
	mockClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, io.ErrUnexpectedEOF
		}),
	}

	notifier := NewDiscordNotifier(
		"https://discord.com/api/webhooks/test",
		WithHTTPClient(mockClient),
		WithAsyncSend(false),
	)

	ctx := context.Background()
	errCtx := ErrorContext{
		ErrorMessage: "test error",
		Timestamp:    time.Now(),
	}

	// Should not panic or crash
	notifier.NotifyError(ctx, errCtx)
}

func TestDiscordNotifier_NotifyError_SkipsWhenDisabled(t *testing.T) {
	requestCount := 0

	mockClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requestCount++
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString("")),
			}, nil
		}),
	}

	// Empty webhook URL - notifier is disabled
	notifier := NewDiscordNotifier(
		"",
		WithHTTPClient(mockClient),
		WithAsyncSend(false),
	)

	ctx := context.Background()
	errCtx := ErrorContext{
		ErrorMessage: "test error",
		Timestamp:    time.Now(),
	}

	notifier.NotifyError(ctx, errCtx)

	if requestCount != 0 {
		t.Errorf("expected 0 requests when notifier is disabled, got %d", requestCount)
	}
}

func TestDiscordNotifier_FormatMessage(t *testing.T) {
	notifier := NewDiscordNotifier("https://discord.com/api/webhooks/test")

	errCtx := ErrorContext{
		ErrorMessage: "test error message",
		StackTrace:   "line 1\nline 2\nline 3",
		RequestID:    "req-123",
		Method:       "POST",
		Path:         "/api/test",
		UserID:       "user-456",
		Timestamp:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	payload := notifier.formatMessage(errCtx)

	embeds, ok := payload["embeds"].([]map[string]interface{})
	if !ok || len(embeds) == 0 {
		t.Fatal("expected embeds in payload")
	}

	embed := embeds[0]

	if embed["title"] != "🚨 Critical Error Detected" {
		t.Errorf("unexpected title: %v", embed["title"])
	}

	if embed["color"] != 0xFF0000 {
		t.Errorf("unexpected color: %v", embed["color"])
	}

	fields, ok := embed["fields"].([]map[string]interface{})
	if !ok {
		t.Fatal("expected fields in embed")
	}

	// Verify error message field
	found := false
	for _, field := range fields {
		if field["name"] == "Error Message" {
			if field["value"] != "test error message" {
				t.Errorf("unexpected error message value: %v", field["value"])
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("error message field not found")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "shorter than max",
			input:  "short",
			maxLen: 10,
			want:   "short",
		},
		{
			name:   "equal to max",
			input:  "exactly10c",
			maxLen: 10,
			want:   "exactly10c",
		},
		{
			name:   "longer than max",
			input:  "this is a very long string",
			maxLen: 10,
			want:   "this is...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateErrorType(t *testing.T) {
	tests := []struct {
		name     string
		errorMsg string
		want     string
	}{
		{
			name:     "short message",
			errorMsg: "short error",
			want:     "short error",
		},
		{
			name:     "long message",
			errorMsg: "this is a very long error message that exceeds fifty characters",
			want:     "this is a very long error message that exceeds fif",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateErrorType(tt.errorMsg)
			if got != tt.want {
				t.Errorf("generateErrorType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	rl := newRateLimiter(100 * time.Millisecond)

	// First call should be allowed
	if !rl.allow("error1") {
		t.Error("first call should be allowed")
	}

	// Immediate second call should be blocked
	if rl.allow("error1") {
		t.Error("immediate second call should be blocked")
	}

	// Wait for rate limit to expire
	time.Sleep(150 * time.Millisecond)

	// Third call should be allowed
	if !rl.allow("error1") {
		t.Error("call after rate limit expiry should be allowed")
	}
}

func TestRateLimiter_DifferentErrorTypes(t *testing.T) {
	rl := newRateLimiter(100 * time.Millisecond)

	// Different error types should each be allowed
	if !rl.allow("error1") {
		t.Error("first error type should be allowed")
	}

	if !rl.allow("error2") {
		t.Error("second error type should be allowed")
	}

	// But same error type should still be rate limited
	if rl.allow("error1") {
		t.Error("same error type should be rate limited")
	}
}

// roundTripFunc is a helper type for mocking HTTP transport
type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

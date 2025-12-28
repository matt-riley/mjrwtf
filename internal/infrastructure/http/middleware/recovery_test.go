package middleware

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/matt-riley/mjrwtf/internal/adapters/notification"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/logging"
	"github.com/rs/zerolog"
)

func TestRecovery_RecoverFromPanic(t *testing.T) {
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "Internal Server Error") {
		t.Errorf("expected error message in response body, got %s", rec.Body.String())
	}
}

func TestRecovery_NoPanic(t *testing.T) {
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if rec.Body.String() != "success" {
		t.Errorf("expected response 'success', got %s", rec.Body.String())
	}
}

func TestRecoveryWithLogger_UsesFallbackLogger(t *testing.T) {
	var logBuf bytes.Buffer
	fallbackLogger := zerolog.New(&logBuf)

	handler := RecoveryWithLogger(fallbackLogger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic with fallback logger")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Verify response status
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	// Verify panic was logged using the fallback logger
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "panic recovered") {
		t.Errorf("expected 'panic recovered' in log output, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "test panic with fallback logger") {
		t.Errorf("expected panic message in log output, got: %s", logOutput)
	}
}

func TestRecoveryWithLogger_UsesContextLogger(t *testing.T) {
	var fallbackBuf bytes.Buffer
	fallbackLogger := zerolog.New(&fallbackBuf)

	var contextBuf bytes.Buffer
	contextLogger := zerolog.New(&contextBuf)

	handler := RecoveryWithLogger(fallbackLogger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic with context logger")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// Add logger to context
	ctx := logging.WithLogger(req.Context(), contextLogger)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Verify response status
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	// Verify panic was logged using the context logger, not the fallback
	contextLogOutput := contextBuf.String()
	if !strings.Contains(contextLogOutput, "panic recovered") {
		t.Errorf("expected 'panic recovered' in context log output, got: %s", contextLogOutput)
	}
	if !strings.Contains(contextLogOutput, "test panic with context logger") {
		t.Errorf("expected panic message in context log output, got: %s", contextLogOutput)
	}

	// Verify fallback logger was not used
	fallbackLogOutput := fallbackBuf.String()
	if fallbackLogOutput != "" {
		t.Errorf("expected fallback logger to be empty when context logger is available, got: %s", fallbackLogOutput)
	}
}

func TestRecoveryWithLogger_LogsPanicDetails(t *testing.T) {
	t.Setenv("LOG_STACK_TRACES", "true")

	var logBuf bytes.Buffer
	logger := zerolog.New(&logBuf)

	handler := RecoveryWithLogger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("detailed panic error")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	logOutput := logBuf.String()

	// Verify panic message is logged
	if !strings.Contains(logOutput, "detailed panic error") {
		t.Errorf("expected panic message in log output, got: %s", logOutput)
	}

	// Verify stack trace is logged
	if !strings.Contains(logOutput, "stack") {
		t.Errorf("expected stack trace in log output, got: %s", logOutput)
	}

	// Verify log level is error
	if !strings.Contains(logOutput, `"level":"error"`) {
		t.Errorf("expected error level in log output, got: %s", logOutput)
	}
}

func TestRecoveryWithLogger_NoPanic(t *testing.T) {
	var logBuf bytes.Buffer
	logger := zerolog.New(&logBuf)

	handler := RecoveryWithLogger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != "success" {
		t.Errorf("expected response 'success', got %s", rec.Body.String())
	}

	// Verify nothing was logged
	if logBuf.String() != "" {
		t.Errorf("expected no log output when no panic, got: %s", logBuf.String())
	}
}

func TestRecoveryWithNotifier_SendsDiscordNotification(t *testing.T) {
	var logBuf bytes.Buffer
	logger := zerolog.New(&logBuf)

	// Track if Discord notification was sent
	notificationSent := false

	mockClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			notificationSent = true
			// Read body to capture context (in real test, would parse JSON)
			_, _ = io.ReadAll(req.Body)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString("")),
			}, nil
		}),
	}

	notifier := notification.NewDiscordNotifier(
		"https://discord.com/api/webhooks/test",
		notification.WithHTTPClient(mockClient),
		notification.WithAsyncSend(false),
	)

	handler := RecoveryWithNotifier(logger, notifier)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic for Discord")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test-path", nil)
	ctx := logging.WithRequestID(req.Context(), "test-req-123")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Verify response status
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	// Verify Discord notification was sent
	if !notificationSent {
		t.Error("expected Discord notification to be sent")
	}

	// Verify panic was also logged locally
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "panic recovered") {
		t.Errorf("expected 'panic recovered' in log output, got: %s", logOutput)
	}
}

func TestRecoveryWithNotifier_WithUserID(t *testing.T) {
	var logBuf bytes.Buffer
	logger := zerolog.New(&logBuf)

	var capturedRequestPath string
	mockClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			capturedRequestPath = req.URL.Path
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString("")),
			}, nil
		}),
	}

	notifier := notification.NewDiscordNotifier(
		"https://discord.com/api/webhooks/test",
		notification.WithHTTPClient(mockClient),
		notification.WithAsyncSend(false),
	)

	handler := RecoveryWithNotifier(logger, notifier)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic with user")
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	// Add user ID to context using the same pattern as auth middleware
	ctx := context.WithValue(req.Context(), UserIDKey, "user-789")
	ctx = logging.WithRequestID(ctx, "req-456")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Verify notification was sent
	if capturedRequestPath != "/api/webhooks/test" {
		t.Error("expected Discord webhook to be called")
	}
}

func TestRecoveryWithNotifier_NoNotifierConfigured(t *testing.T) {
	var logBuf bytes.Buffer
	logger := zerolog.New(&logBuf)

	// nil notifier - should not crash
	handler := RecoveryWithNotifier(logger, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic without notifier")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should still recover and respond
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	// Should still log
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "panic recovered") {
		t.Errorf("expected 'panic recovered' in log output")
	}
}

func TestRecoveryWithNotifier_DisabledNotifier(t *testing.T) {
	var logBuf bytes.Buffer
	logger := zerolog.New(&logBuf)

	notificationSent := false
	mockClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			notificationSent = true
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString("")),
			}, nil
		}),
	}

	// Empty webhook URL - notifier is disabled
	notifier := notification.NewDiscordNotifier(
		"",
		notification.WithHTTPClient(mockClient),
		notification.WithAsyncSend(false),
	)

	handler := RecoveryWithNotifier(logger, notifier)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic with disabled notifier")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should recover normally
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	// Should NOT send notification
	if notificationSent {
		t.Error("expected no Discord notification when notifier is disabled")
	}

	// Should still log
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "panic recovered") {
		t.Errorf("expected 'panic recovered' in log output")
	}
}

func TestRecoveryWithLogger_StackTraceDisabled(t *testing.T) {
	t.Setenv("LOG_STACK_TRACES", "false")

	var logBuf bytes.Buffer
	logger := zerolog.New(&logBuf)

	handler := RecoveryWithLogger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("panic without stack")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	logOutput := logBuf.String()
	if strings.Contains(logOutput, `"stack"`) {
		t.Errorf("expected no stack trace in log output when LOG_STACK_TRACES=false, got: %s", logOutput)
	}
}

func TestRecoveryWithNotifier_StackTraceDisabled(t *testing.T) {
	t.Setenv("LOG_STACK_TRACES", "false")

	var logBuf bytes.Buffer
	logger := zerolog.New(&logBuf)

	var payload string
	mockClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			b, _ := io.ReadAll(req.Body)
			payload = string(b)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString("")),
			}, nil
		}),
	}

	notifier := notification.NewDiscordNotifier(
		"https://discord.com/api/webhooks/test",
		notification.WithHTTPClient(mockClient),
		notification.WithAsyncSend(false),
	)

	handler := RecoveryWithNotifier(logger, notifier)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("panic without stack")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if strings.Contains(payload, "Stack Trace") {
		t.Errorf("expected no stack trace field in Discord payload when LOG_STACK_TRACES=false, got: %s", payload)
	}
}

func TestRecovery_RepanicsOnErrAbortHandler(t *testing.T) {
	h := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(http.ErrAbortHandler)
	}))

	defer func() {
		rec := recover()
		if rec != http.ErrAbortHandler {
			t.Errorf("expected panic %v, got %v", http.ErrAbortHandler, rec)
		}
	}()

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/test", nil))
}

// roundTripFunc is a helper type for mocking HTTP transport
type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

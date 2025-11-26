package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

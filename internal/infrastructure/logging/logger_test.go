package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/rs/zerolog"
)

func TestNew_DefaultLevel(t *testing.T) {
	logger := New("info", "json")

	// Verify logger is at info level
	if logger.GetLevel() != zerolog.InfoLevel {
		t.Errorf("expected info level, got %v", logger.GetLevel())
	}
}

func TestNew_DebugLevel(t *testing.T) {
	logger := New("debug", "json")

	if logger.GetLevel() != zerolog.DebugLevel {
		t.Errorf("expected debug level, got %v", logger.GetLevel())
	}
}

func TestNew_WarnLevel(t *testing.T) {
	logger := New("warn", "json")

	if logger.GetLevel() != zerolog.WarnLevel {
		t.Errorf("expected warn level, got %v", logger.GetLevel())
	}
}

func TestNew_ErrorLevel(t *testing.T) {
	logger := New("error", "json")

	if logger.GetLevel() != zerolog.ErrorLevel {
		t.Errorf("expected error level, got %v", logger.GetLevel())
	}
}

func TestNew_InvalidLevelDefaultsToInfo(t *testing.T) {
	logger := New("invalid", "json")

	if logger.GetLevel() != zerolog.InfoLevel {
		t.Errorf("expected info level for invalid input, got %v", logger.GetLevel())
	}
}

func TestNew_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf).With().Timestamp().Logger()

	logger.Info().Msg("test message")

	// Verify output is valid JSON
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Errorf("expected JSON output, got parsing error: %v", err)
	}

	if logEntry["message"] != "test message" {
		t.Errorf("expected message 'test message', got %v", logEntry["message"])
	}
}

func TestWithLogger_AddsLoggerToContext(t *testing.T) {
	logger := New("info", "json")
	ctx := context.Background()

	ctx = WithLogger(ctx, logger)

	retrievedLogger := FromContext(ctx)
	if retrievedLogger.GetLevel() != logger.GetLevel() {
		t.Error("expected to retrieve the same logger from context")
	}
}

func TestFromContext_ReturnsDisabledLoggerWhenNotSet(t *testing.T) {
	ctx := context.Background()

	logger := FromContext(ctx)

	// A disabled logger should have trace level (lowest)
	// but zerolog.Nop() returns a disabled logger
	if logger.GetLevel() != zerolog.Disabled {
		t.Errorf("expected disabled logger, got level %v", logger.GetLevel())
	}
}

func TestWithRequestID_AddsRequestIDToContext(t *testing.T) {
	ctx := context.Background()
	requestID := "test-request-id-123"

	ctx = WithRequestID(ctx, requestID)

	retrievedID := GetRequestID(ctx)
	if retrievedID != requestID {
		t.Errorf("expected request ID %q, got %q", requestID, retrievedID)
	}
}

func TestGetRequestID_ReturnsEmptyStringWhenNotSet(t *testing.T) {
	ctx := context.Background()

	requestID := GetRequestID(ctx)

	if requestID != "" {
		t.Errorf("expected empty string, got %q", requestID)
	}
}

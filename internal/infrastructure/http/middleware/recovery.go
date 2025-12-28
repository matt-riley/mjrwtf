package middleware

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"github.com/matt-riley/mjrwtf/internal/adapters/notification"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/logging"
	"github.com/rs/zerolog"
)

var (
	stackTracesOnce   sync.Once
	stackTracesCached bool
)

func isAbortHandlerPanic(rec any) bool {
	if rec == http.ErrAbortHandler {
		return true
	}
	err, ok := rec.(error)
	return ok && errors.Is(err, http.ErrAbortHandler)
}

func stackTracesEnabled() bool {
	stackTracesOnce.Do(func() {
		stackTracesCached = readStackTracesEnabled()
	})
	return stackTracesCached
}

func readStackTracesEnabled() bool {
	v, ok := os.LookupEnv("LOG_STACK_TRACES")
	if !ok {
		return true // backward compatible default
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		zl := zerolog.New(os.Stderr)
		zl.Warn().
			Err(err).
			Str("env", "LOG_STACK_TRACES").
			Str("value", v).
			Msg("invalid boolean value for LOG_STACK_TRACES; valid values are: true, false, 1, 0, t, f, T, F, TRUE, FALSE, True, False; defaulting to true")
		return true
	}
	return b
}

// recoveryWriter wraps http.ResponseWriter to track if headers have been written
type recoveryWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (rw *recoveryWriter) WriteHeader(statusCode int) {
	if !rw.wroteHeader {
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(statusCode)
	}
}

func (rw *recoveryWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// Flush implements http.Flusher
func (rw *recoveryWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack implements http.Hijacker
func (rw *recoveryWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// Push implements http.Pusher
func (rw *recoveryWriter) Push(target string, opts *http.PushOptions) error {
	if p, ok := rw.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}

// RecoveryWithLogger returns a recovery middleware that uses the provided logger.
// This ensures panics are logged even if the context-based logger isn't available.
func RecoveryWithLogger(logger zerolog.Logger) func(http.Handler) http.Handler {
	return RecoveryWithNotifier(logger, nil)
}

// RecoveryWithNotifier returns a recovery middleware that uses the provided logger and Discord notifier.
// This ensures panics are logged and optionally sent to Discord for critical errors.
func RecoveryWithNotifier(logger zerolog.Logger, notifier *notification.DiscordNotifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := &recoveryWriter{ResponseWriter: w}
			defer func() {
				if rec := recover(); rec != nil {
					if isAbortHandlerPanic(rec) {
						panic(rec)
					}

					// Try to get logger from context first, fall back to provided logger
					ctxLogger := logging.FromContext(r.Context())
					if ctxLogger.GetLevel() == zerolog.Disabled {
						ctxLogger = logger
					}

					includeStack := stackTracesEnabled()
					stackTrace := ""
					if includeStack {
						stackTrace = string(debug.Stack())
					}
					errorMsg := fmt.Sprintf("%v", rec)

					evt := ctxLogger.Error().
						Str("panic_type", fmt.Sprintf("%T", rec)).
						Str("panic", errorMsg)
					if includeStack {
						evt = evt.Str("stack", stackTrace)
					}
					evt.Msg("panic recovered")

					// Send notification to Discord if notifier is configured
					if notifier != nil && notifier.IsEnabled() {
						userID, _ := GetUserID(r.Context())
						errCtx := notification.ErrorContext{
							ErrorMessage: errorMsg,
							StackTrace:   stackTrace,
							RequestID:    logging.GetRequestID(r.Context()),
							Method:       r.Method,
							Path:         r.URL.Path,
							UserID:       userID,
							Timestamp:    time.Now(),
						}
						notifier.NotifyError(r.Context(), errCtx)
					}

					if !rw.wroteHeader {
						rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
						rw.WriteHeader(http.StatusInternalServerError)
						_, _ = rw.Write([]byte("Internal Server Error"))
					}
					// If headers already sent, we can't write error response
				}
			}()
			next.ServeHTTP(rw, r)
		})
	}
}

// Recovery recovers from panics and logs the error using structured logging.
// It uses the logger from context if available, otherwise logs nothing.
// For full logging support, use RecoveryWithLogger instead.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &recoveryWriter{ResponseWriter: w}
		defer func() {
			if rec := recover(); rec != nil {
				if isAbortHandlerPanic(rec) {
					panic(rec)
				}

				logger := logging.FromContext(r.Context())

				includeStack := stackTracesEnabled()
				stackTrace := ""
				if includeStack {
					stackTrace = string(debug.Stack())
				}
				errorMsg := fmt.Sprintf("%v", rec)

				evt := logger.Error().
					Str("panic_type", fmt.Sprintf("%T", rec)).
					Str("panic", errorMsg)
				if includeStack {
					evt = evt.Str("stack", stackTrace)
				}
				evt.Msg("panic recovered")

				if !rw.wroteHeader {
					rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
					rw.WriteHeader(http.StatusInternalServerError)
					_, _ = rw.Write([]byte("Internal Server Error"))
				}
				// If headers already sent, we can't write error response
			}
		}()
		next.ServeHTTP(rw, r)
	})
}

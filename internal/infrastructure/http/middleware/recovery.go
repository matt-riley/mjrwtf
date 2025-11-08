package middleware

import (
	"bufio"
	"log"
	"net"
	"net/http"
	"runtime/debug"
)

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

// Recovery recovers from panics and logs the error
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &recoveryWriter{ResponseWriter: w}
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic recovered: %v", err)
				// Only log stack trace in detailed format if needed for debugging
				// For production, consider logging to a secure location
				log.Printf("stack trace:\n%s", debug.Stack())

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

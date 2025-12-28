// Package middleware contains HTTP middleware for auth, sessions, logging, recovery, security headers, metrics, and rate limiting.
//
// # Panic / recover policy
//
// In general, application code should return errors for expected runtime failures
// (bad input, IO/network issues, database errors, etc.) and translate those errors
// into appropriate HTTP responses.
//
// Panics are reserved for unrecoverable programmer errors (violated invariants,
// impossible states) where continuing could corrupt state or hide a serious bug.
//
// Recovery middleware is a last line of defense. It:
//   - recovers unexpected panics and returns a 500 response if possible
//   - logs only minimal request context (method + path) and avoids headers,
//     cookies, query strings, and full URLs
//   - optionally notifies Discord (same minimal context)
//
// Special case: panics equal to http.ErrAbortHandler are re-panicked to preserve
// net/http semantics (abort without noisy logging).
//
// Stack traces can be disabled via LOG_STACK_TRACES=false (default: true). When
// disabled, neither logs nor Discord notifications include stack traces.
package middleware

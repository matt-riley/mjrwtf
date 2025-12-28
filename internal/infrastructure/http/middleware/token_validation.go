package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
)

// ValidateTokenConstantTime checks whether token matches any of validTokens.
// It avoids early-exit across the configured token list to reduce timing signal.
//
// Note: total runtime is proportional to len(validTokens), which may leak the
// configured token count via timing. In practice this is typically 1–2 tokens
// during rotations, and network jitter dominates.
//
// Returns:
//   - match: true if token matches one of validTokens
//   - configured: true if validTokens is non-empty
func ValidateTokenConstantTime(token string, validTokens []string) (match bool, configured bool) {
	if len(validTokens) == 0 {
		h := sha256.Sum256([]byte(token))
		var zero [sha256.Size]byte
		_ = subtle.ConstantTimeCompare(h[:], zero[:])
		return false, false
	}

	m := 0
	for _, t := range validTokens {
		m |= subtle.ConstantTimeCompare([]byte(token), []byte(t))
	}
	return m == 1, true
}

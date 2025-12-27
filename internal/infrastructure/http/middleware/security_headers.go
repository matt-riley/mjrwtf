package middleware

import "net/http"

// SecurityHeaders returns a middleware that sets security-related HTTP headers
// to protect against common web vulnerabilities.
//
// Headers set:
//   - X-Content-Type-Options: nosniff (prevents MIME sniffing)
//   - X-Frame-Options: DENY (prevents clickjacking)
//   - Referrer-Policy: strict-origin-when-cross-origin (controls referrer info)
//   - Content-Security-Policy: baseline policy for templ-based pages
//   - Strict-Transport-Security: HSTS (only if enableHSTS is true)
//
// The CSP policy allows:
//   - Scripts from self and Tailwind CDN (unsafe-inline for templ inline scripts)
//   - Styles from self (unsafe-inline for templ inline styles)
//   - Images from self and data URIs
//   - Frame ancestors: none (same as X-Frame-Options: DENY)
//
// Parameters:
//   - enableHSTS: Set to true to enable Strict-Transport-Security header.
//     Only enable this when the application is behind TLS/HTTPS.
func SecurityHeaders(enableHSTS bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prevent MIME type sniffing
			w.Header().Set("X-Content-Type-Options", "nosniff")

			// Prevent clickjacking
			w.Header().Set("X-Frame-Options", "DENY")

			// Control referrer information
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Content Security Policy
			// Note: 'unsafe-inline' is required for templ-generated inline scripts/styles
			// and Tailwind CDN script
			w.Header().Set("Content-Security-Policy",
				"default-src 'self'; "+
					"script-src 'self' 'unsafe-inline' https://cdn.tailwindcss.com; "+
					"style-src 'self' 'unsafe-inline'; "+
					"img-src 'self' data:; "+
					"font-src 'self'; "+
					"connect-src 'self'; "+
					"frame-ancestors 'none'")

			// HSTS (only when behind TLS - configurable via environment)
			if enableHSTS {
				w.Header().Set("Strict-Transport-Security",
					"max-age=31536000; includeSubDomains")
			}

			next.ServeHTTP(w, r)
		})
	}
}

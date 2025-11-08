package middleware

import (
	"log"
	"net/http"
	"runtime/debug"
)

// Recovery recovers from panics and logs the error
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic recovered: %v\nstack trace:\n%s", err, debug.Stack())
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Internal Server Error"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

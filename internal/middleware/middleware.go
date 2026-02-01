package middleware

import (
	"bytes"
	"golb/internal/config"
	"net/http"
)

// Auth validates the secret-token header when EnableAuth is true.
func Auth(cfg *config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cfg.EnableAuth {
			token := r.Header.Get("secret-token")
			if token != cfg.AuthToken {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// cachedWriter captures response body and status for Redis cache (used by cache.go).
type cachedWriter struct {
	http.ResponseWriter
	body *bytes.Buffer
	code int
}

func (w *cachedWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *cachedWriter) WriteHeader(statusCode int) {
	w.code = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

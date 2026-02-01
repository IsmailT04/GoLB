package middleware

import (
	"bytes"
	"context"
	"golb/internal/config"
	"log/slog"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache uses Redis for distributed response caching (GET only).
// cachedWriter is defined in middleware.go.
func Cache(cfg *config.Config, rdb *redis.Client, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !cfg.EnableCache || r.Method != http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.Background()
		url := r.URL.String()
		keyBody := "cache:body:" + url
		keyType := "cache:type:" + url

		cachedBody, err := rdb.Get(ctx, keyBody).Bytes()
		if err == nil {
			contentType, _ := rdb.Get(ctx, keyType).Result()
			w.Header().Set("Content-Type", contentType)
			w.Header().Set("X-Cache", "HIT")
			w.Write(cachedBody)
			return
		}

		writer := &cachedWriter{
			ResponseWriter: w,
			body:           &bytes.Buffer{},
			code:           http.StatusOK,
		}

		next.ServeHTTP(writer, r)

		if writer.code == http.StatusOK {
			go func() {
				pipe := rdb.Pipeline()
				pipe.Set(context.Background(), keyBody, writer.body.Bytes(), 1*time.Minute)
				pipe.Set(context.Background(), keyType, writer.Header().Get("Content-Type"), 1*time.Minute)
				_, execErr := pipe.Exec(context.Background())
				if execErr != nil {
					slog.Error("failed to save to redis cache", "error", execErr)
				}
			}()
		}
	})
}

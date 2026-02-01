package middleware

import (
	"context"
	"golb/internal/config"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimit uses Redis for distributed rate limiting (fixed window per IP per minute).
func RateLimit(cfg *config.Config, rdb *redis.Client, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !cfg.EnableRateLimit {
			next.ServeHTTP(w, r)
			return
		}

		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		currentMinute := time.Now().Format("2006-01-02T15:04")
		key := "ratelimit:" + ip + ":" + currentMinute

		ctx := context.Background()

		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			// Fail open: allow request if Redis is down
			next.ServeHTTP(w, r)
			return
		}

		if count == 1 {
			rdb.Expire(ctx, key, 1*time.Minute)
		}

		if int(count) > cfg.RateLimitPerMin {
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(cfg.RateLimitPerMin))
			w.Header().Set("X-RateLimit-Remaining", "0")
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

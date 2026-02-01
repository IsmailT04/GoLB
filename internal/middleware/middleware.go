package middleware

import (
    "bytes"
    "golb/internal/config"
    "log/slog"
    "net"
    "net/http"
    "sync"
    "time"
)

// AUTHENTICATION MIDDLEWARE
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

// RATE LIMITING MIDDLEWARE (Fixed Window Counter)
type RateLimiter struct {
    mu      sync.Mutex
    clients map[string]int
    reset   time.Time
    limit   int
}

var limiter *RateLimiter

func NewRateLimiter(limit int) *RateLimiter {
    return &RateLimiter{
        clients: make(map[string]int),
        reset:   time.Now().Add(time.Minute),
        limit:   limit,
    }
}

func RateLimit(cfg *config.Config, next http.Handler) http.Handler {
    // Initialize singleton if needed
    if limiter == nil {
        limiter = NewRateLimiter(cfg.RateLimitPerMin)
    }

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if cfg.EnableRateLimit {
            ip, _, _ := net.SplitHostPort(r.RemoteAddr)
            
            limiter.mu.Lock()
            // Reset window if minute passed
            if time.Now().After(limiter.reset) {
                limiter.clients = make(map[string]int)
                limiter.reset = time.Now().Add(time.Minute)
            }

            limiter.clients[ip]++
            count := limiter.clients[ip]
            limiter.mu.Unlock()

            if count > limiter.limit {
                http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
                return
            }
        }
        next.ServeHTTP(w, r)
    })
}

// 3. CACHING MIDDLEWARE
type CacheEntry struct {
    Body        []byte
    ContentType string
    Expiry      time.Time
}

var (
    cache = make(map[string]CacheEntry)
    cMu   sync.RWMutex
)

// Custom ResponseWriter to capture the response body
type cachedWriter struct {
    http.ResponseWriter
    body *bytes.Buffer
    code int
}

func (w *cachedWriter) Write(b []byte) (int, error) {
    w.body.Write(b) // Capture data
    return w.ResponseWriter.Write(b) // Pass through to client
}

func (w *cachedWriter) WriteHeader(statusCode int) {
    w.code = statusCode
    w.ResponseWriter.WriteHeader(statusCode)
}

func Cache(cfg *config.Config, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Only cache GET requests if enabled
        if !cfg.EnableCache || r.Method != http.MethodGet {
            next.ServeHTTP(w, r)
            return
        }

        url := r.URL.String()

        // Check Cache
        cMu.RLock()
        entry, found := cache[url]
        cMu.RUnlock()

        if found && time.Now().Before(entry.Expiry) {
            slog.Info("serving from cache", "url", url)
            w.Header().Set("Content-Type", entry.ContentType)
            w.Header().Set("X-Cache", "HIT")
            w.Write(entry.Body)
            return
        }

        // Cache Miss: Wrap the writer to capture response
        writer := &cachedWriter{
            ResponseWriter: w,
            body:           &bytes.Buffer{},
            code:           http.StatusOK,
        }

        next.ServeHTTP(writer, r)

        // Save to Cache if status is 200 OK
        if writer.code == http.StatusOK {
            cMu.Lock()
            cache[url] = CacheEntry{
                Body:        writer.body.Bytes(),
                ContentType: writer.Header().Get("Content-Type"),
                Expiry:      time.Now().Add(1 * time.Minute), // Cache for 1 min
            }
            cMu.Unlock()
        }
    })
}
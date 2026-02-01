package backend

import (
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

type Backend struct {
	Url                    *url.URL
	alive                  bool         // Live or dead (circuit open/closed)
	lock                   sync.RWMutex
	connections            int64        // Storing active requests
	Weight                 int          // Static config
	CurrentWeight          int          // Dynamic "Score"
	proxy                  *httputil.ReverseProxy
	Logger                 *slog.Logger // optional; if nil, uses slog.Default()
	consecutiveFailures     int64        // atomic: current failure count
	maxConsecutiveFailures  int64        // threshold to trip circuit breaker
}

// NewBackend initializes a backend with a reverse proxy. logger may be nil (uses slog.Default()).
// maxConsecutiveFailures is the circuit breaker threshold; if <= 0, defaults to 3.
func NewBackend(u *url.URL, weight int, logger *slog.Logger, maxConsecutiveFailures int64) *Backend {
	proxy := httputil.NewSingleHostReverseProxy(u)

	// Custom Transport to handle timeouts (prevents hanging connections)
	proxy.Transport = &http.Transport{
		MaxIdleConns:    100,
		IdleConnTimeout: 90 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	if logger == nil {
		logger = slog.Default()
	}
	if maxConsecutiveFailures <= 0 {
		maxConsecutiveFailures = 3
	}

	b := &Backend{
		Url:                   u,
		alive:                 true,
		Weight:                weight,
		proxy:                 proxy,
		Logger:                logger,
		maxConsecutiveFailures: maxConsecutiveFailures,
	}

	// ErrorHandler: increment failure count, trip circuit if threshold reached
	b.proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		count := atomic.AddInt64(&b.consecutiveFailures, 1)
		b.Logger.Warn("proxy request error", "host", b.Url.Host, "error", err, "consecutive_failures", count)

		if count >= b.maxConsecutiveFailures {
			b.SetAlive(false)
			b.Logger.Warn("circuit breaker tripped: marking backend as dead", "url", b.Url.String(), "failures", count)
		}
		w.WriteHeader(http.StatusBadGateway)
	}

	// ModifyResponse: reset failure count on success (close / reset the "streak")
	b.proxy.ModifyResponse = func(*http.Response) error {
		atomic.StoreInt64(&b.consecutiveFailures, 0)
		return nil
	}

	return b
}

func (b *Backend) SetAlive(alive bool) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.alive = alive
}

func (b *Backend) IsAlive() bool {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.alive
}

func (b *Backend) GetActiveConnections() int64 {
	return atomic.LoadInt64(&b.connections)
}

// ResetConsecutiveFailures resets the failure counter (e.g. when health check restores the backend).
func (b *Backend) ResetConsecutiveFailures() {
	atomic.StoreInt64(&b.consecutiveFailures, 0)
}

// ServeHTTP satisfies the http.Handler interface and tracks connections
func (b *Backend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt64(&b.connections, 1)
	defer atomic.AddInt64(&b.connections, -1)

	b.proxy.ServeHTTP(w, r)
}

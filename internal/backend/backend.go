package backend

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
)

type Backend struct {
	Url         *url.URL
	alive       bool         //Live or dead
	lock        sync.RWMutex //lock to handle write and read of status
	connections int64        // Storing active requests
	Weight      int          // For Weighted Round Robin
	proxy       *httputil.ReverseProxy
}

// NewBackend initializes a backend with a reverse proxy
func NewBackend(u *url.URL) *Backend {
	return &Backend{
		Url:    u,
		alive:  true,
		Weight: 1,
		proxy:  httputil.NewSingleHostReverseProxy(u),
	}
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

// ServeHTTP satisfies the http.Handler interface and tracks connections
func (b *Backend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt64(&b.connections, 1)
	defer atomic.AddInt64(&b.connections, -1)

	b.proxy.ServeHTTP(w, r)
}

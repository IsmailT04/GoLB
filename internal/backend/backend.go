package backend

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

type Backend struct {
	Url           *url.URL
	alive         bool         //Live or dead
	lock          sync.RWMutex //lock to handle write and read of status
	connections   int64        // Storing active requests
	Weight        int          // Static config
	CurrentWeight int          //Dynamic "Score"
	proxy         *httputil.ReverseProxy
}

// NewBackend initializes a backend with a reverse proxy
func NewBackend(u *url.URL, weight int) *Backend {
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

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("[%s] Proxy Request Error: %v", u.Host, err)
		w.WriteHeader(http.StatusBadGateway)
	}

	return &Backend{
		Url:    u,
		alive:  true,
		Weight: weight,
		proxy:  proxy,
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

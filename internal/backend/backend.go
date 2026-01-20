package backend

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
)

type Backend struct {
	Url   *url.URL
	alive bool         //Live or dead
	lock  sync.RWMutex //lock to handle write and read of status
	proxy *httputil.ReverseProxy
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

// NewBackend initializes a backend with a reverse proxy
func NewBackend(u *url.URL) *Backend {
	return &Backend{
		Url:   u,
		alive: true,
		proxy: httputil.NewSingleHostReverseProxy(u),
	}
}

// ServeHTTP makes Backend satisfy the http.Handler interface
func (b *Backend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b.proxy.ServeHTTP(w, r)
}

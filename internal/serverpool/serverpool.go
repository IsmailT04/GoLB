package serverpool

import (
	"golb/internal/backend"
	"log"
	"net"
	"sync/atomic"
	"time"
)

type ServerPool struct {
	Backends []*backend.Backend
	current  uint64
}

func (s *ServerPool) AddBackend(backend *backend.Backend) {
	s.Backends = append(s.Backends, backend)
}

func (s *ServerPool) NextIndex() int {
	return int(atomic.AddUint64(&s.current, 1) % uint64(len(s.Backends)))
}

func (s *ServerPool) GetNextPeer() *backend.Backend {
	next := s.NextIndex()
	l := len(s.Backends) + next

	for i := next; i < l; i++ {
		idx := i % len(s.Backends)

		if s.Backends[idx].IsAlive() {
			if i != next {
				atomic.StoreUint64(&s.current, uint64(idx))
			}
			return s.Backends[idx]
		}

	}
	return nil
}

func (s *ServerPool) HealthCheck() {
	for _, backend := range s.Backends {
		url := backend.Url
		conn, err := net.DialTimeout("tcp", url.Host, 2*time.Second)
		if err != nil {
			backend.SetAlive(false)
		} else {
			conn.Close()
			backend.SetAlive(true)
		}
	}
}

func (s *ServerPool) StartHealthCheck() {
	t := time.NewTicker(20 * time.Second)

	for {
		log.Println("Starting health check...")
		s.HealthCheck()
		<-t.C
	}
}

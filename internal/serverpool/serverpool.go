package serverpool

import (
	"golb/internal/backend"
	"sync/atomic"
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

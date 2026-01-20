package serverpool

import (
	"golb/internal/backend"
	"sync/atomic"
)

// RoundRobin Strategy
type RoundRobin struct {
	current uint64
}

func (r *RoundRobin) GetNextPeer(backends []*backend.Backend) *backend.Backend {
	next := int(atomic.AddUint64(&r.current, 1) % uint64(len(backends)))


	l := len(backends) + next
	for i := next; i < l; i++ {
		idx := i % len(backends) // Wrap around

		if backends[idx].IsAlive() {
			if i != next {
				atomic.StoreUint64(&r.current, uint64(idx))
			}
			return backends[idx]
		}
	}
	return nil
}

// LeastConnections Strategy
type LeastConnections struct{}

func (l *LeastConnections) GetNextPeer(backends []*backend.Backend) *backend.Backend {
	var best *backend.Backend
	min := int64(-1)

	for _, b := range backends {
		if !b.IsAlive() {
			continue
		}

		conn := b.GetActiveConnections()

		// Logic: If active connections < min, or if it's the first one we see (min == -1)
		if min == -1 || conn < min {
			min = conn
			best = b
		}
	}
	return best
}

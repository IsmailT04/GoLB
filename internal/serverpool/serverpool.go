package serverpool

import (
	"golb/internal/backend"
	"log/slog"
	"net"
	"time"
)

type ServerPool struct {
	Backends []*backend.Backend
	Strategy BalancingStrategy
	Logger   *slog.Logger // optional; if nil, uses slog.Default()
}

func (s *ServerPool) AddBackend(backend *backend.Backend) {
	s.Backends = append(s.Backends, backend)
}

// GetNextPeer delegates the decision to the strategy
func (s *ServerPool) GetNextPeer() *backend.Backend {
	return s.Strategy.GetNextPeer(s.Backends)
}

// HealthCheck loops through backends and pings them. When a backend is alive, its failure counter is reset.
func (s *ServerPool) HealthCheck() {
	for _, be := range s.Backends {
		conn, err := net.DialTimeout("tcp", be.Url.Host, 2*time.Second)
		if err != nil {
			be.SetAlive(false)
		} else {
			conn.Close()
			be.SetAlive(true)
			be.ResetConsecutiveFailures() // sync with circuit breaker: clean slate when health check restores
		}
	}
}

// StartHealthCheck runs in a loop
func (s *ServerPool) StartHealthCheck() {
	log := s.Logger
	if log == nil {
		log = slog.Default()
	}
	t := time.NewTicker(20 * time.Second)
	for {
		log.Debug("starting health check")
		s.HealthCheck()
		<-t.C
	}
}

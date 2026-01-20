package serverpool

import (
	"golb/internal/backend"
	"log"
	"net"
	"time"
)

type ServerPool struct {
	Backends []*backend.Backend
	Strategy BalancingStrategy
}

func (s *ServerPool) AddBackend(backend *backend.Backend) {
	s.Backends = append(s.Backends, backend)
}

// GetNextPeer delegates the decision to the strategy
func (s *ServerPool) GetNextPeer() *backend.Backend {
	return s.Strategy.GetNextPeer(s.Backends)
}

// HealthCheck loops through backends and pings them
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

// StartHealthCheck runs in a loop
func (s *ServerPool) StartHealthCheck() {
	t := time.NewTicker(20 * time.Second)
	for {
		log.Println("Starting health check...")
		s.HealthCheck()
		<-t.C
	}
}

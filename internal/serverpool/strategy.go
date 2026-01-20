package serverpool

import "golb/internal/backend"

// BalancingStrategy defines how to select the next backend
type BalancingStrategy interface {
	GetNextPeer(backends []*backend.Backend) *backend.Backend
}

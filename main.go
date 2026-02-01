package main

import (
	"fmt"
	"golb/internal/backend"
	"golb/internal/config"
	"golb/internal/middleware"
	"golb/internal/serverpool"
	"log"
	"net/http"
	"net/url"
)

func main() {
	// 1. Load Configuration
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// 2. Initialize Server Pool
	pool := &serverpool.ServerPool{}

	// 3. Set Strategy
	switch cfg.Strategy {
	case "round-robin":
		pool.Strategy = &serverpool.RoundRobin{}
	case "weighted-round-robin":
		pool.Strategy = &serverpool.WeightedRoundRobin{}
	case "least-connections":
		pool.Strategy = &serverpool.LeastConnections{}
	default:
		log.Fatalf("Invalid strategy in config: %s", cfg.Strategy)
	}

	// 4. Configure Backends
	for _, b := range cfg.Backends {
		u, err := url.Parse(b.URL)
		if err != nil {
			log.Fatalf("Failed to parse backend URL %s: %v", b.URL, err)
		}

		// Create backend with specific weight from config
		backendInstance := backend.NewBackend(u, b.Weight)
		pool.AddBackend(backendInstance)
		log.Printf("Added backend: %s [Weight: %d]", b.URL, b.Weight)
	}

	// 5. Start Health Checks
	go pool.StartHealthCheck()

	// 6. Define the Core Load Balancing Logic
	// This is the "final destination" handler
	lbHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peer := pool.GetNextPeer()
		if peer != nil {
			peer.ServeHTTP(w, r)
			return
		}
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	})

	// 7. Chain the Middleware
	// Flow: Request -> RateLimit -> Auth -> Cache -> LBHandler
	handler := middleware.RateLimit(cfg,
		middleware.Auth(cfg,
			middleware.Cache(cfg, lbHandler),
		),
	)

	// 8. Start Server
	server := http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.LBPort),
		Handler: handler, // Use the wrapped handler
	}

	log.Printf("Load Balancer started on port %d using %s strategy", cfg.LBPort, cfg.Strategy)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

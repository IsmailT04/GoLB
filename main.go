package main

import (
	"golb/internal/backend"
	"golb/internal/serverpool"
	"log"
	"net/http"
	"net/url"
)

func main() {
	// Create a ServerPool instance
	pool := &serverpool.ServerPool{}

	// Parse backend URLs
	backendURLs := []string{
		"http://localhost:8081",
		"http://localhost:8082",
		"http://localhost:8083",
	}

	// Add backends to the pool
	for _, urlStr := range backendURLs {
		u, err := url.Parse(urlStr)
		if err != nil {
			log.Fatalf("Failed to parse URL %s: %v", urlStr, err)
		}

		backendInstance := backend.NewBackend(u)
		pool.AddBackend(backendInstance)
		log.Printf("Added backend: %s", urlStr)
	}

	// Update the http.HandleFunc
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("LB Received request: %s %s", r.Method, r.URL.Path)

		// Get next peer from the pool
		peer := pool.GetNextPeer()

		// If all servers are dead, return 503 Service Unavailable
		if peer == nil {
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
			return
		}

		// Forward the request to the selected peer
		peer.ServeHTTP(w, r)
	})

	log.Println("Load Balancer started on :8080")

	// Start the Load Balancer server
	log.Fatal(http.ListenAndServe(":8080", nil))
}

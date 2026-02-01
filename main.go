package main

import (
	"context"
	"fmt"
	"golb/internal/backend"
	"golb/internal/config"
	"golb/internal/middleware"
	"golb/internal/serverpool"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// 1. Structured logger (JSON for production aggregation)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	// 2. Load Configuration (file + env overrides)
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// 3. Initialize Server Pool and inject logger
	pool := &serverpool.ServerPool{Logger: logger}

	// 4. Set Strategy
	switch cfg.Strategy {
	case "round-robin":
		pool.Strategy = &serverpool.RoundRobin{}
	case "weighted-round-robin":
		pool.Strategy = &serverpool.WeightedRoundRobin{}
	case "least-connections":
		pool.Strategy = &serverpool.LeastConnections{}
	default:
		logger.Error("invalid strategy in config", "strategy", cfg.Strategy)
		os.Exit(1)
	}

	// 5. Configure Backends
	for _, b := range cfg.Backends {
		u, err := url.Parse(b.URL)
		if err != nil {
			logger.Error("failed to parse backend URL", "url", b.URL, "error", err)
			os.Exit(1)
		}
		backendInstance := backend.NewBackend(u, b.Weight, logger, b.MaxConsecutiveFailures)
		pool.AddBackend(backendInstance)
		logger.Info("added backend", "url", b.URL, "weight", b.Weight)
	}

	// 6. Start Health Checks
	go pool.StartHealthCheck()

	// 7. Define the Core Load Balancing Logic
	lbHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peer := pool.GetNextPeer()
		if peer != nil {
			peer.ServeHTTP(w, r)
			return
		}
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	})

	// 8. Chain the Middleware (Metrics wraps the chain for the main server)
	handler := middleware.Metrics(
		middleware.RateLimit(cfg,
			middleware.Auth(cfg,
				middleware.Cache(cfg, lbHandler),
			),
		),
	)

	// Metrics server on separate port (isolates admin from user traffic)
	go func() {
		metricsServer := http.Server{
			Addr:    ":9090",
			Handler: middleware.MetricsHandler(),
		}
		logger.Info("metrics server starting", "port", 9090)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("metrics server error", "error", err)
		}
	}()
	// 9. Server with timeouts (anti-Slowloris) and MaxHeaderBytes
	server := http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.LBPort),
		Handler:           handler,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	// 10. Start server in a goroutine (HTTP or HTTPS)
	go func() {
		var serveErr error
		if cfg.CertFile != "" && cfg.KeyFile != "" {
			serveErr = server.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile)
		} else {
			serveErr = server.ListenAndServe()
		}
		if serveErr != nil && serveErr != http.ErrServerClosed {
			logger.Error("server error", "error", serveErr)
			os.Exit(1)
		}
	}()

	scheme := "http"
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		scheme = "https"
	}
	logger.Info("load balancer started", "port", cfg.LBPort, "strategy", cfg.Strategy, "scheme", scheme)

	// 11. Signal channel for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	logger.Info("shutdown signal received, stopping server")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server shutdown error", "error", err)
	}
	logger.Info("server exited")
}

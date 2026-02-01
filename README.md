# GoLB - High Performance Load Balancer

GoLB (Go Load Balancer) is a lightweight, concurrent, and production-ready Layer 7 Load Balancer built in Go. It supports active health checks, circuit breaking, distributed rate limiting and caching (Redis), TLS, graceful shutdown, and structured logging.

## Features

- **Balancing Strategies**
  - `round-robin`: Distributes traffic sequentially.
  - `weighted-round-robin`: Distributes traffic by configured weights.
  - `least-connections`: Sends traffic to the backend with the fewest active requests.
- **Active Health Checks**: TCP ping every 20s; dead backends are skipped until they recover.
- **Circuit Breaker**: Per-backend consecutive failure threshold (`max_consecutive_failures`); trips and marks backend dead, resets on success or when health check restores it.
- **Middleware**
  - **Auth**: Optional header-based auth (`secret-token`).
  - **Rate Limit**: Distributed (Redis) fixed-window rate limit per IP per minute; fail-open if Redis is down.
  - **Cache**: Distributed (Redis) GET response cache; 1-minute TTL.
  - **Metrics**: Prometheus metrics; exposed on a separate admin port (default `:9090`).
- **Production Hardening**: Server timeouts (Read/Write/Idle/ReadHeader), MaxHeaderBytes, graceful shutdown (SIGINT/SIGTERM), structured JSON logging (slog).
- **TLS**: Optional HTTPS via `cert_file` / `key_file` (or `CERT_FILE` / `KEY_FILE` env).
- **Config**: YAML config with env overrides (12-factor style).

## Requirements

- **Go** 1.21+ (for `log/slog`).
- **Redis**: Used for distributed rate limiting and caching. Must be reachable at `redis_addr` (default `localhost:6379`).

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/golb.git
   cd golb
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Ensure Redis is running (e.g. `localhost:6379`), or set `REDIS_ADDR`.

## Usage

### 1. Configure

Copy `config.example.yaml` to `config.yaml` and adjust. Example:

```yaml
lb_port: 8080
strategy: "weighted-round-robin"

enable_auth: true
auth_token: "my-secret"

enable_ratelimit: true
rate_limit_per_min: 5

enable_cache: true
redis_addr: "localhost:6379"

backends:
  - url: "http://localhost:8081"
    weight: 1
    max_consecutive_failures: 3
  - url: "http://localhost:8082"
    weight: 3
    max_consecutive_failures: 3
  - url: "http://localhost:8083"
    weight: 1
    max_consecutive_failures: 3
```

**Environment overrides**: `LB_PORT`, `AUTH_TOKEN`, `CERT_FILE`, `KEY_FILE`, `GOLB_STRATEGY`, `REDIS_ADDR`, `REDIS_PASSWORD`, `REDIS_DB`.

### 2. Start the Load Balancer

```bash
go run main.go
```

The main server listens on `lb_port` (default 8080). The metrics server listens on `:9090` (Prometheus scrape target).

### 3. Test with Dummy Backends

```bash
# In a separate terminal
cd backend
./run_backends.sh
```

Then send requests to `http://localhost:8080`. Use header `secret-token: my-secret` when auth is enabled.

### 4. Metrics

Prometheus metrics are served on port **9090** (e.g. `http://localhost:9090/metrics`). Keep this port internal (e.g. firewall or separate admin network).

## Architecture

### Project Structure

```
golb/
├── internal/
│   ├── backend/         # Backend, reverse proxy, circuit breaker
│   ├── config/          # YAML + env config loader
│   ├── middleware/      # Auth, RateLimit (Redis), Cache (Redis), Metrics
│   └── serverpool/      # Pool, health checks, strategies
├── backend/             # Dummy servers and scripts for testing
├── config.yaml          # Main configuration (see config.example.yaml)
└── main.go              # Entry point, Redis init, server + metrics server
```

### Request Flow

1. **Metrics** (optional instrumentation).
2. **RateLimit** (Redis): per-IP, per-minute; fail-open on Redis errors.
3. **Auth**: validate `secret-token` if `enable_auth` is true.
4. **Cache** (Redis): GET only; cache hit returns stored body + content-type.
5. **Load balancer**: select backend via strategy, proxy request; circuit breaker can mark backend dead after N consecutive proxy failures.

### Health Checks & Circuit Breaker

- A background goroutine TCP-pings each backend every 20s. Failed pings mark the backend dead; successful pings mark it alive and **reset its consecutive failure count**.
- The proxy **ErrorHandler** increments a per-backend failure counter on each proxy error. When the count reaches `max_consecutive_failures`, the backend is marked dead (circuit tripped).
- **ModifyResponse** resets the failure counter on successful backend responses, so a single success clears the streak.

## License



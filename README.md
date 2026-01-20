# GoLB - High Performance Load Balancer

GoLB (Go Load Balancer) is a lightweight, concurrent, and production-ready Layer 7 Load Balancer built in Go. It supports active health checks, modular load balancing strategies, and dynamic configuration via YAML.

## Features

- **Balancing Strategies**:
  - `round-robin`: Distributes traffic sequentially.
  - `weighted-round-robin`: Distributes traffic based on server capacity (weights).
  - `least-connections`: Sends traffic to the server with the fewest active requests.
- **Active Health Checks**: Automatically removes dead servers from the pool and re-adds them when they recover.
- **Production Grade**: Includes timeouts, connection pooling, and error handling.
- **Configurable**: Simple YAML configuration.

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/golb.git
   cd golb
   ```

2. Download dependencies:
   ```bash
   go mod tidy
   ```

## Usage

### 1. Configure the Load Balancer
Edit `config.yaml` in the root directory:

```yaml
lb_port: 8080
strategy: "weighted-round-robin" 
backends:
  - url: "http://localhost:8081"
    weight: 1
  - url: "http://localhost:8082"
    weight: 3
  - url: "http://localhost:8083"
    weight: 1
```

### 2. Start the Load Balancer
```bash
go run main.go
```

### 3. Test with Dummy Backends
We provide a script to spin up multiple dummy backend servers for testing.

```bash
# In a separate terminal
cd backend
./run_backends.sh
```

## Architecture

### Project Structure
```
golb/
├── internal/
│   ├── backend/       # Backend struct & ReverseProxy logic
│   ├── config/        # YAML Configuration loader
│   └── serverpool/    # Pool management & Strategies
├── backend/           # Dummy server & scripts for testing
├── config.yaml        # Main configuration
└── main.go            # Entry point
```

### Health Checks
The Load Balancer runs a background routine that attempts to establish a TCP connection with every backend every 20 seconds.
- If a connection fails, the backend is marked `dead` and skipped during routing.
- If a connection succeeds on a previously dead backend, it is marked `alive` and returned to rotation.

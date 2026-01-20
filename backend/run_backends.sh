#!/bin/bash

# Script to run multiple backend instances on different ports

# Array of ports to run backends on
PORTS=(8081 8082 8083)


# Function to cleanup background processes on exit
cleanup() {
    echo ""
    echo "Stopping all backend servers..."
    kill $(jobs -p) 2>/dev/null
    exit
}

# Trap SIGINT and SIGTERM to cleanup
trap cleanup SIGINT SIGTERM

# Start each backend on a different port
for port in "${PORTS[@]}"; do
    echo "Starting backend on port $port..."
    go run deneme.go -port="$port" &
    sleep 1  # Small delay to ensure proper startup
done

echo ""
echo "All backend servers started!"
echo "Backends running on ports: ${PORTS[*]}"
echo "Press Ctrl+C to stop all servers"
echo ""

# Wait for all background jobs
wait

#!/bin/bash

# Stop any running Consul container
echo "Stopping Consul container..."
docker stop consul-server || true
docker rm consul-server || true

# Start Consul with the updated configuration
echo "Starting Consul with host network mode..."
docker-compose up -d

# Wait for Consul to start
echo "Waiting for Consul to start..."
sleep 5

# Kill any running node service
echo "Stopping node service..."
pkill -f "./bin/node" || true

# Build and start the node service
echo "Building and starting node service..."
make build-node
make run-node

echo "Services restarted successfully!" 
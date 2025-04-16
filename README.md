# Blockchain Event Gateway

A decentralized blockchain event crawling system with a gateway service that routes clients to nodes crawling specific blockchain chains.

## System Architecture

The system consists of three main components:

1. **Gateway Service**: Routes clients to nodes crawling specific blockchain chains
2. **Node Service**: Crawls blockchain chains and streams events to clients
3. **Client**: Connects to the gateway to find a node, then streams blockchain events

## Components

### Gateway Service

The gateway service:
- Uses Consul for service discovery
- Routes clients to nodes based on chain ID
- Implements round-robin load balancing per chain
- Filters nodes based on a whitelist
- Returns node addresses in the format hostname:port

### Node Service

The node service:
- Registers with Consul using tags like `chain:ethereum` or `chain:bitcoin`
- Exposes a gRPC streaming service for clients
- Simulates streaming blockchain events
- Includes health checks for Consul

### Client

The client:
- Connects to the gateway to find a node for a specific chain
- Connects directly to the node for event streaming
- Processes and displays blockchain events

## Prerequisites

- Go 1.16 or later
- Consul (for service discovery)
- Make (for building)

## Building

```bash
# Build all components
make build

# Build individual components
make build-gateway
make build-node
make build-client
```

## Running

### Using the Demo Script

The easiest way to run the entire system is using the demo script:

```bash
./scripts/run_demo.sh
```

This will:
1. Start Consul if not already running
2. Build all components
3. Start the gateway service
4. Start nodes for Ethereum and Bitcoin
5. Start clients for Ethereum and Bitcoin
6. Display logs in separate files

### Running Components Individually

#### Gateway

```bash
./bin/gateway -c config/config.yaml
```

#### Node

```bash
./bin/node -c config/config.yaml
```

#### Client

```bash
./bin/client -c config/config.yaml -chain ethereum
```

## Configuration

Configuration is managed through YAML files in the `config` directory:

- `config.yaml`: Default configuration
- `bitcoin_node.yaml`: Configuration for Bitcoin node

You can customize the configuration by editing these files or by setting environment variables with the `EVENT_CATCHER_` prefix.

## How It Works

1. Nodes register with Consul using tags like `chain:ethereum` or `chain:bitcoin`
2. Clients request a node for a specific chain from the gateway
3. The gateway finds healthy nodes for the requested chain using Consul
4. The gateway returns a node address to the client
5. The client connects directly to the node and streams blockchain events

## License

MIT 
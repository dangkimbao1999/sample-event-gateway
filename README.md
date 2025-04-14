# Event Catcher Gateway

A gateway service for routing clients to nodes with specific data, enabling real-time streaming while prioritizing consistency and persistence.

## Features

- gRPC-based communication
- Service discovery using Consul
- Persistent data-to-node mapping
- Real-time data streaming
- Automatic failover and health checking
- Offset-based streaming resume
- Configuration management with Viper
- Command-line interface with Cobra

## Prerequisites

- Go 1.21 or later
- Protocol Buffers compiler (protoc)
- Consul server
- Go plugins for protoc:
  ```bash
  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
  ```

## Project Structure

```
.
├── cmd/
│   ├── gateway/         # Gateway service entry point
│   └── node/           # Node service entry point
├── config/             # Configuration management
│   └── config.yaml     # Configuration file
├── gateway/            # Gateway service implementation
├── proto/              # Protocol Buffer definitions
├── examples/
│   └── client/        # Example client implementation
├── scripts/           # Utility scripts
└── Makefile          # Build and development tasks
```

## Building

1. Generate Protocol Buffer code:
   ```bash
   make proto
   ```

2. Build all components:
   ```bash
   make build build-node build-client
   ```

## Running the System

1. Start a Consul server (if not already running):
   ```bash
   consul agent -dev
   ```

2. Set up the data-to-node mapping in Consul:
   ```bash
   chmod +x scripts/setup-consul.sh
   ./scripts/setup-consul.sh --data-id=test-data --node-id=node1
   ```

3. Start the gateway service:
   ```bash
   make run
   ```

4. Start a node service:
   ```bash
   make run-node
   ```

5. Run the example client:
   ```bash
   make run-client
   ```

## Configuration

The services can be configured using a YAML configuration file or environment variables:

### Configuration File

The default configuration file is located at `config/config.yaml`. You can specify a different configuration file using the `--config` flag:

```bash
./bin/gateway --config=/path/to/config.yaml
```

### Environment Variables

Environment variables can be used to override configuration values. The environment variables are prefixed with `EVENT_CATCHER_` and use underscores instead of dots:

```
EVENT_CATCHER_GATEWAY_HOST=0.0.0.0
EVENT_CATCHER_GATEWAY_PORT=50051
EVENT_CATCHER_CONSUL_HOST=localhost
EVENT_CATCHER_CONSUL_PORT=8500
EVENT_CATCHER_CONSUL_KV_PREFIX=streaming/data/
```

### Configuration Options

#### Gateway Service
- `gateway.host`: Host for the gateway service (default: 0.0.0.0)
- `gateway.port`: Port for the gateway service (default: 50051)

#### Node Service
- `node.id`: Unique identifier for the node (default: node1)
- `node.port`: Port to listen on (default: 50052)
- `node.health_check.path`: Health check path (default: /health)
- `node.health_check.interval`: Health check interval (default: 10s)
- `node.health_check.timeout`: Health check timeout (default: 5s)

#### Consul
- `consul.host`: Consul server host (default: localhost)
- `consul.port`: Consul server port (default: 8500)
- `consul.kv_prefix`: Prefix for Consul KV store (default: streaming/data/)

#### Logging
- `log.level`: Log level (default: info)
- `log.format`: Log format (default: text)

## Development

To clean the build artifacts:
```bash
make clean
```

## License

MIT License 
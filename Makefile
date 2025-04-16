.PHONY: proto build build-gateway build-node build-client run run-gateway run-node run-client run-demo clean

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GATEWAY_BINARY=gateway
NODE_BINARY=node
CLIENT_BINARY=client
PROTO_DIR=proto
GO_OUT_DIR=proto
CONFIG_DIR=config
BIN_DIR=bin

# Protobuf parameters
PROTOC=protoc
PROTO_INCLUDES=-I$(PROTO_DIR)
PROTO_GO_OPT=--go_out=./proto --go_opt=paths=source_relative
PROTO_GRPC_OPT=--go-grpc_out=./proto --go-grpc_opt=paths=source_relative

# Create bin directory if it doesn't exist
$(shell mkdir -p $(BIN_DIR))

proto:
	$(PROTOC) $(PROTO_INCLUDES) $(PROTO_GO_OPT) $(PROTO_GRPC_OPT) $(PROTO_DIR)/*.proto

build: proto build-gateway build-node build-client

build-gateway: proto
	$(GOBUILD) -o $(BIN_DIR)/$(GATEWAY_BINARY) cmd/gateway/main.go

build-node: proto
	$(GOBUILD) -o $(BIN_DIR)/$(NODE_BINARY) cmd/node/main.go

build-client: proto
	$(GOBUILD) -o $(BIN_DIR)/$(CLIENT_BINARY) cmd/client/main.go

run-gateway: build-gateway
	./$(BIN_DIR)/$(GATEWAY_BINARY) -config $(CONFIG_DIR)/config.yaml

run-node: build-node
	./$(BIN_DIR)/$(NODE_BINARY) -config $(CONFIG_DIR)/config.yaml

run-client: build-client
	./$(BIN_DIR)/$(CLIENT_BINARY) -config $(CONFIG_DIR)/config.yaml -chain ethereum

run-demo:
	./scripts/run_demo.sh

clean:
	$(GOCLEAN)
	rm -f $(BIN_DIR)/$(GATEWAY_BINARY)
	rm -f $(BIN_DIR)/$(NODE_BINARY)
	rm -f $(BIN_DIR)/$(CLIENT_BINARY)
	rm -f $(GO_OUT_DIR)/*.pb.go 
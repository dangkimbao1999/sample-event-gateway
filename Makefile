.PHONY: proto build build-node build-client run run-node run-client clean

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

# Protobuf parameters
PROTOC=protoc
PROTO_INCLUDES=-I$(PROTO_DIR)
PROTO_GO_OPT=--go_out=./proto --go_opt=paths=source_relative
PROTO_GRPC_OPT=--go-grpc_out=./proto --go-grpc_opt=paths=source_relative

proto:
	$(PROTOC) $(PROTO_INCLUDES) $(PROTO_GO_OPT) $(PROTO_GRPC_OPT) $(PROTO_DIR)/*.proto

build: proto
	$(GOBUILD) -o bin/$(GATEWAY_BINARY) cmd/gateway/main.go

build-node: proto
	$(GOBUILD) -o bin/$(NODE_BINARY) cmd/node/main.go

build-client: proto
	$(GOBUILD) -o bin/$(CLIENT_BINARY) examples/client/main.go

run: build
	./bin/$(GATEWAY_BINARY) --config=$(CONFIG_DIR)/config.yaml

run-node: build-node
	./bin/$(NODE_BINARY) --config=$(CONFIG_DIR)/config.yaml

run-client: build-client
	./bin/$(CLIENT_BINARY) --config=$(CONFIG_DIR)/config.yaml --data-id=test-data

clean:
	$(GOCLEAN)
	rm -f bin/$(GATEWAY_BINARY)
	rm -f bin/$(NODE_BINARY)
	rm -f bin/$(CLIENT_BINARY)
	rm -f $(GO_OUT_DIR)/*.pb.go 
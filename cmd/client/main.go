package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"event-catcher-gateway/config"
	pb "event-catcher-gateway/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	configPath = flag.String("config", "config/config.yaml", "path to config file")
	chainID    = flag.String("chain", "polygon", "blockchain chain ID to connect to")
)

func main() {
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to the gateway
	gatewayAddr := cfg.GetGatewayAddr()
	log.Printf("Connecting to gateway at %s", gatewayAddr)

	gatewayConn, err := grpc.Dial(gatewayAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to gateway: %v", err)
	}
	defer gatewayConn.Close()

	// Create gateway client
	gatewayClient := pb.NewGatewayServiceClient(gatewayConn)

	// Request a node for the specified chain
	ctx := context.Background()
	resp, err := gatewayClient.GetNodeForChain(ctx, &pb.GetNodeRequest{
		ChainId: "polygon",
	})
	if err != nil {
		log.Fatalf("Failed to get node for chain %s: %v", *chainID, err)
	}

	if resp.ErrorMessage != "" {
		log.Fatalf("Gateway error: %s", resp.ErrorMessage)
	}

	log.Printf("Gateway returned node address: %s", resp.NodeAddress)

	// Connect to the node
	nodeConn, err := grpc.Dial(resp.NodeAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to node: %v", err)
	}
	defer nodeConn.Close()

	// Create node client
	nodeClient := pb.NewNodeClient(nodeConn)

	// Set up context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Start streaming blockchain events
	stream, err := nodeClient.StreamData(ctx, &pb.StreamRequest{
		DataId: *chainID,
		Offset: 0,
	})
	if err != nil {
		log.Fatalf("Failed to start streaming: %v", err)
	}

	log.Printf("Started streaming blockchain events for chain: %s", *chainID)

	// Receive and process events
	for {
		chunk, err := stream.Recv()
		if err != nil {
			log.Printf("Stream ended: %v", err)
			return
		}

		log.Printf("Received blockchain event: %s (offset: %d, timestamp: %d)",
			chunk.Data, chunk.Offset, chunk.Timestamp)
	}
}

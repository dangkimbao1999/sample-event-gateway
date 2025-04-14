package main

import (
	"context"
	"io"
	"log"
	"os"

	"event-catcher-gateway/config"
	pb "event-catcher-gateway/proto"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	configPath string
	dataID     string
	rootCmd    = &cobra.Command{
		Use:   "client",
		Short: "Event Catcher Client",
		Long:  `A client for streaming data from nodes.`,
		RunE:  runClient,
	}
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "path to config file")
	rootCmd.PersistentFlags().StringVarP(&dataID, "data-id", "d", "test-data", "data ID to stream")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func runClient(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return err
	}

	// Connect to gateway service
	gatewayAddr := cfg.GetGatewayAddr()
	gatewayConn, err := grpc.Dial(gatewayAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to gateway: %v", err)
	}
	defer gatewayConn.Close()

	// Create gateway client
	gatewayClient := pb.NewGatewayClient(gatewayConn)

	// Get node information from gateway
	ctx := context.Background()
	nodeResp, err := gatewayClient.GetNodeForData(ctx, &pb.GetNodeRequest{
		DataId: dataID,
	})
	if err != nil {
		log.Fatalf("Failed to get node information: %v", err)
	}

	log.Printf("Found node %s at %s", nodeResp.NodeId, nodeResp.NodeAddress)

	// Connect to node service
	nodeConn, err := grpc.Dial(nodeResp.NodeAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to node: %v", err)
	}
	defer nodeConn.Close()

	// Create node client
	nodeClient := pb.NewNodeClient(nodeConn)

	// Start streaming data
	stream, err := nodeClient.StreamData(ctx, &pb.StreamRequest{
		DataId: dataID,
		Offset: 0,
	})
	if err != nil {
		log.Fatalf("Failed to start streaming: %v", err)
	}

	// Receive data chunks
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			log.Println("Stream ended")
			break
		}
		if err != nil {
			log.Printf("Error receiving chunk: %v", err)
			break
		}

		log.Printf("Received chunk: offset=%d, timestamp=%d, data=%s",
			chunk.Offset,
			chunk.Timestamp,
			string(chunk.Data))
	}

	return nil
}

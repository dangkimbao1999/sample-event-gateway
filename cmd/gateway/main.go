package main

import (
	"log"
	"net"
	"os"

	"event-catcher-gateway/config"
	"event-catcher-gateway/gateway"
	pb "event-catcher-gateway/proto"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var (
	configPath string
	rootCmd    = &cobra.Command{
		Use:   "gateway",
		Short: "Event Catcher Gateway Service",
		Long:  `A gateway service for routing clients to nodes with specific data, enabling real-time streaming.`,
		RunE:  runGateway,
	}
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "path to config file")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func runGateway(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return err
	}

	// Create the gateway service
	gatewayService, err := gateway.NewService(cfg.GetConsulAddr(), cfg.Consul.KVPrefix)
	if err != nil {
		log.Fatalf("Failed to create gateway service: %v", err)
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()
	pb.RegisterGatewayServer(grpcServer, gatewayService)

	// Start listening
	lis, err := net.Listen("tcp", cfg.GetGatewayAddr())
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Printf("Gateway service listening on %s", cfg.GetGatewayAddr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}

	return nil
}

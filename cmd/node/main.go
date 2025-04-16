package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"event-catcher-gateway/config"
	pb "event-catcher-gateway/proto"

	"github.com/hashicorp/consul/api"
	"google.golang.org/grpc"
)

var (
	configPath = flag.String("config", "config/config.yaml", "path to config file")
)

type nodeServer struct {
	pb.UnimplementedNodeServer
	nodeID string
	chains []string
}

func (s *nodeServer) StreamData(req *pb.StreamRequest, stream pb.Node_StreamDataServer) error {
	log.Printf("Starting data stream for data ID: %s from offset: %d", req.DataId, req.Offset)

	// Simulate streaming blockchain events
	for i := req.Offset; ; i++ {
		// Create a data chunk with blockchain event data
		chunk := &pb.DataChunk{
			Data:      []byte(fmt.Sprintf("Blockchain event %d for %s", i, req.DataId)),
			Offset:    i,
			Timestamp: time.Now().Unix(),
			DataId:    req.DataId,
		}

		log.Printf("Sending blockchain event: %s", chunk.Data)

		// Send the chunk
		if err := stream.Send(chunk); err != nil {
			return err
		}

		// Simulate some processing time
		time.Sleep(100 * time.Millisecond)
	}
}

func main() {
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create gRPC server
	lis, err := net.Listen("tcp", cfg.GetNodeAddr())
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	nodeServer := &nodeServer{
		nodeID: cfg.Node.ID,
		chains: cfg.Node.Chains,
	}
	pb.RegisterNodeServer(grpcServer, nodeServer)

	// Create HTTP server for health checks
	httpServer := &http.Server{
		Addr: fmt.Sprintf(":%d", cfg.Node.HealthCheck.Port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == cfg.Node.HealthCheck.Path {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			} else {
				http.NotFound(w, r)
			}
		}),
	}

	// Start HTTP server first to ensure it's running before registering with Consul
	go func() {
		log.Printf("Starting HTTP server for health checks on port %d", cfg.Node.HealthCheck.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to serve HTTP: %v", err)
		}
	}()

	// Wait a moment to ensure the HTTP server is started
	time.Sleep(1 * time.Second)

	// Register with Consul
	consulConfig := api.DefaultConfig()
	consulConfig.Address = cfg.GetConsulAddr()
	consulClient, err := api.NewClient(consulConfig)
	if err != nil {
		log.Fatalf("Failed to create Consul client: %v", err)
	}

	// Get the local IP address for the health check
	localIP := "127.0.0.1" // Use localhost for Consul registration

	// Log the IP address being used
	log.Printf("Using IP for Consul registration: %s", localIP)

	// Create tags for each chain this node handles
	var tags []string
	for _, chain := range cfg.Node.Chains {
		tags = append(tags, fmt.Sprintf("chain:%s", chain))
	}

	// Register service
	registration := &api.AgentServiceRegistration{
		ID:      cfg.Node.ID,
		Name:    "blockchain-node",
		Port:    cfg.Node.Port,
		Address: localIP,
		Check: &api.AgentServiceCheck{
			HTTP:     fmt.Sprintf("http://%s:%d%s", localIP, cfg.Node.HealthCheck.Port, cfg.Node.HealthCheck.Path),
			Interval: cfg.Node.HealthCheck.Interval,
			Timeout:  cfg.Node.HealthCheck.Timeout,
		},
		Tags: tags,
	}

	if err := consulClient.Agent().ServiceRegister(registration); err != nil {
		log.Fatalf("Failed to register service: %v", err)
	}
	log.Printf("Successfully registered service with Consul: %s, chains: %v", cfg.Node.ID, cfg.Node.Chains)

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		cancel()
	}()

	// Start gRPC server
	go func() {
		log.Printf("Starting gRPC server on %s", cfg.GetNodeAddr())
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	<-ctx.Done()

	// Graceful shutdown
	log.Println("Shutting down servers...")

	// Shutdown HTTP server
	httpCtx, httpCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer httpCancel()
	if err := httpServer.Shutdown(httpCtx); err != nil {
		log.Printf("Failed to shutdown HTTP server: %v", err)
	}

	// Shutdown gRPC server
	grpcServer.GracefulStop()

	// Deregister from Consul
	if err := consulClient.Agent().ServiceDeregister(cfg.Node.ID); err != nil {
		log.Printf("Failed to deregister service: %v", err)
	}
}

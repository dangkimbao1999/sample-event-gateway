package gateway

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/hashicorp/consul/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "event-catcher-gateway/proto"
)

// Service implements the GatewayService gRPC service
type Service struct {
	pb.UnimplementedGatewayServiceServer
	consulClient *api.Client
	// Track the next node index for each chain for round-robin selection
	chainIndices map[string]*atomic.Uint64
	indicesMu    sync.RWMutex
	// Whitelist of allowed node IDs
	whitelist map[string]bool
	mu        sync.RWMutex
}

// NewService creates a new gateway service instance
func NewService(consulAddr string) (*Service, error) {
	config := api.DefaultConfig()
	config.Address = consulAddr

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Consul client: %v", err)
	}

	// Initialize whitelist with hardcoded nodes
	whitelist := map[string]bool{
		"node1": true,
		"node2": true,
		"node3": true,
	}

	return &Service{
		consulClient: client,
		whitelist:    whitelist,
		chainIndices: make(map[string]*atomic.Uint64),
	}, nil
}

// GetNodeForChain implements the GetNodeForChain RPC method
func (s *Service) GetNodeForChain(ctx context.Context, req *pb.GetNodeRequest) (*pb.GetNodeResponse, error) {
	// Query Consul for healthy services with the chain tag
	tag := fmt.Sprintf("chain:%s", req.ChainId)
	services, _, err := s.consulClient.Health().Service("blockchain-node", tag, true, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query Consul service catalog: %v", err)
	}

	// Filter services by whitelist
	var whitelistedServices []*api.ServiceEntry
	for _, service := range services {
		s.mu.RLock()
		isWhitelisted := s.whitelist[service.Service.ID]
		s.mu.RUnlock()

		if isWhitelisted {
			whitelistedServices = append(whitelistedServices, service)
		}
	}

	if len(whitelistedServices) == 0 {
		return &pb.GetNodeResponse{
			ErrorMessage: fmt.Sprintf("no healthy nodes available for chain: %s", req.ChainId),
		}, nil
	}

	// Get the next node index for round-robin selection
	s.indicesMu.RLock()
	index, exists := s.chainIndices[req.ChainId]
	s.indicesMu.RUnlock()

	if !exists {
		// Initialize the index if it doesn't exist
		s.indicesMu.Lock()
		index = &atomic.Uint64{}
		s.chainIndices[req.ChainId] = index
		s.indicesMu.Unlock()
	}

	// Get the next node index using atomic operations
	nodeIndex := index.Add(1) % uint64(len(whitelistedServices))
	selectedService := whitelistedServices[nodeIndex]

	// Format the node address
	nodeAddress := fmt.Sprintf("%s:%d", selectedService.Service.Address, selectedService.Service.Port)

	log.Printf("Selected node %s at %s for chain %s (using round-robin, index: %d/%d)",
		selectedService.Service.ID, nodeAddress, req.ChainId, nodeIndex+1, len(whitelistedServices))

	return &pb.GetNodeResponse{
		NodeAddress: nodeAddress,
	}, nil
}

package gateway

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/consul/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "event-catcher-gateway/proto"
)

// Service implements the Gateway gRPC service
type Service struct {
	pb.UnimplementedGatewayServer
	consulClient *api.Client
	kvPrefix     string
}

// NewService creates a new gateway service instance
func NewService(consulAddr, kvPrefix string) (*Service, error) {
	config := api.DefaultConfig()
	config.Address = consulAddr

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Consul client: %v", err)
	}

	return &Service{
		consulClient: client,
		kvPrefix:     kvPrefix,
	}, nil
}

// GetNodeForData implements the GetNodeForData RPC method
func (s *Service) GetNodeForData(ctx context.Context, req *pb.GetNodeRequest) (*pb.GetNodeResponse, error) {
	// Query Consul KV store for the node ID associated with the data ID
	kvPair, _, err := s.consulClient.KV().Get(s.kvPrefix+req.DataId, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query Consul KV store: %v", err)
	}
	if kvPair == nil {
		return nil, status.Errorf(codes.NotFound, "no node found for data ID: %s", req.DataId)
	}

	nodeID := string(kvPair.Value)

	// Get the healthy service instance for this node
	services, _, err := s.consulClient.Health().Service("streaming-node", "", true, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query Consul service catalog: %v", err)
	}

	// Find the specific node in the healthy services
	var nodeAddress string
	for _, service := range services {
		if service.Service.ID == nodeID {
			nodeAddress = fmt.Sprintf("%s:%d", service.Service.Address, service.Service.Port)
			break
		}
	}

	if nodeAddress == "" {
		return nil, status.Errorf(codes.Unavailable, "node %s is not healthy or not found", nodeID)
	}

	log.Printf("Found node %s at %s for data ID %s", nodeID, nodeAddress, req.DataId)
	return &pb.GetNodeResponse{
		NodeAddress: nodeAddress,
		NodeId:      nodeID,
	}, nil
}

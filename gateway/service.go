package gateway

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"

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
	whitelist    map[string]bool
	mu           sync.RWMutex
	// Track the next node index for each data ID for round-robin selection
	nodeIndices map[string]*atomic.Uint64
	indicesMu   sync.RWMutex
}

// NewService creates a new gateway service instance
func NewService(consulAddr, kvPrefix string) (*Service, error) {
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
		kvPrefix:     kvPrefix,
		whitelist:    whitelist,
		nodeIndices:  make(map[string]*atomic.Uint64),
	}, nil
}

// RegisterNode implements the RegisterNode RPC method
func (s *Service) RegisterNode(ctx context.Context, req *pb.RegisterNodeRequest) (*pb.RegisterNodeResponse, error) {
	// Check if the node is in the whitelist
	s.mu.RLock()
	isWhitelisted := s.whitelist[req.NodeId]
	s.mu.RUnlock()

	if !isWhitelisted {
		log.Printf("Node registration rejected: %s is not in the whitelist", req.NodeId)
		return nil, status.Errorf(codes.PermissionDenied, "node %s is not in the whitelist", req.NodeId)
	}

	// Get existing nodes for this data ID
	kvPair, _, err := s.consulClient.KV().Get(s.kvPrefix+req.DataId, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query Consul KV store: %v", err)
	}

	var nodeList []string
	if kvPair != nil {
		// Parse existing nodes
		nodeList = parseNodeList(string(kvPair.Value))
	}

	// Check if node is already registered
	for _, node := range nodeList {
		if node == req.NodeId {
			log.Printf("Node %s already registered for data ID %s", req.NodeId, req.DataId)
			return &pb.RegisterNodeResponse{
				Success: true,
				Message: fmt.Sprintf("Node %s already registered for data ID %s", req.NodeId, req.DataId),
			}, nil
		}
	}

	// Add the new node to the list
	nodeList = append(nodeList, req.NodeId)

	// Store the updated node list in Consul
	_, err = s.consulClient.KV().Put(&api.KVPair{
		Key:   s.kvPrefix + req.DataId,
		Value: []byte(serializeNodeList(nodeList)),
	}, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to store node mapping: %v", err)
	}

	// Initialize the node index for round-robin if it doesn't exist
	s.indicesMu.Lock()
	if _, exists := s.nodeIndices[req.DataId]; !exists {
		s.nodeIndices[req.DataId] = &atomic.Uint64{}
	}
	s.indicesMu.Unlock()

	log.Printf("Node %s registered for data ID %s (total nodes: %d)", req.NodeId, req.DataId, len(nodeList))
	return &pb.RegisterNodeResponse{
		Success: true,
		Message: fmt.Sprintf("Node %s successfully registered for data ID %s (total nodes: %d)", req.NodeId, req.DataId, len(nodeList)),
	}, nil
}

// GetNodeForData implements the GetNodeForData RPC method
func (s *Service) GetNodeForData(ctx context.Context, req *pb.GetNodeRequest) (*pb.GetNodeResponse, error) {
	// Query Consul KV store for the node IDs associated with the data ID
	kvPair, _, err := s.consulClient.KV().Get(s.kvPrefix+req.DataId, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query Consul KV store: %v", err)
	}
	if kvPair == nil {
		return nil, status.Errorf(codes.NotFound, "no nodes found for data ID: %s", req.DataId)
	}

	// Parse the list of nodes
	nodeList := parseNodeList(string(kvPair.Value))
	if len(nodeList) == 0 {
		return nil, status.Errorf(codes.NotFound, "no nodes found for data ID: %s", req.DataId)
	}

	// Get the next node index for round-robin selection
	s.indicesMu.RLock()
	index, exists := s.nodeIndices[req.DataId]
	s.indicesMu.RUnlock()

	if !exists {
		// Initialize the index if it doesn't exist
		s.indicesMu.Lock()
		index = &atomic.Uint64{}
		s.nodeIndices[req.DataId] = index
		s.indicesMu.Unlock()
	}

	// Get the next node index using atomic operations
	nodeIndex := index.Add(1) % uint64(len(nodeList))
	nodeID := nodeList[nodeIndex]

	// Check if the node is in the whitelist
	s.mu.RLock()
	isWhitelisted := s.whitelist[nodeID]
	s.mu.RUnlock()

	if !isWhitelisted {
		return nil, status.Errorf(codes.PermissionDenied, "node %s is not in the whitelist", nodeID)
	}

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

	log.Printf("Selected node %s at %s for data ID %s (using round-robin, index: %d/%d)",
		nodeID, nodeAddress, req.DataId, nodeIndex+1, len(nodeList))
	return &pb.GetNodeResponse{
		NodeAddress: nodeAddress,
		NodeId:      nodeID,
	}, nil
}

// Helper function to parse a comma-separated list of node IDs
func parseNodeList(value string) []string {
	if value == "" {
		return []string{}
	}

	// Split by comma and trim whitespace
	nodes := strings.Split(value, ",")
	for i, node := range nodes {
		nodes[i] = strings.TrimSpace(node)
	}
	return nodes
}

// Helper function to serialize a list of node IDs to a comma-separated string
func serializeNodeList(nodes []string) string {
	return strings.Join(nodes, ",")
}

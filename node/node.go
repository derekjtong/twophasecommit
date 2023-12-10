package node

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	"path/filepath"
	"sync"
)

type Node struct {
	Name string
	Addr string
	Type string
	// Coordinator Related
	c_participantClients map[string]*rpc.Client
	// Participant Related
	p_coordinatorClient *rpc.Client
	fileLock            sync.Mutex
}

func NewParticipant(addr string, name string) (*Node, error) {
	return &Node{
		Name: name,
		Addr: addr,
		Type: "Participant",
	}, nil
}

func NewCoordinator(addr string) (*Node, error) {
	return &Node{
		Name: "C",
		Addr: addr,
		Type: "Coordinator",
	}, nil
}

func (n *Node) Print(msg string) {
	fmt.Printf("[%s-%s]: %s\n", n.Type, n.Name, msg)
}

func (n *Node) Start() {
	n.Print(fmt.Sprintf("Starting %s-%s on %s", n.Type, n.Name, n.Addr))

	// Check and create node_data directory
	nodeDataDir := "node_data"
	if _, err := os.Stat(nodeDataDir); os.IsNotExist(err) {
		err := os.Mkdir(nodeDataDir, 0755) // Create directory with read/write/execute for user, and read/execute for group and others
		if err != nil {
			n.Print(fmt.Sprintf("Error creating data directory: %v", err))
			return
		}
	}

	// Create a data file for the node
	filename := fmt.Sprintf("%s-%s.data", n.Type, n.Name) // Change extension based on your data format
	filepath := filepath.Join(nodeDataDir, filename)
	file, err := os.Create(filepath)
	if err != nil {
		n.Print(fmt.Sprintf("Error creating data file: %v", err))
		return
	}
	_, writeErr := file.WriteString("0")
	if writeErr != nil {
		n.Print(fmt.Sprintf("Error writing '0' to file: %v", writeErr))
		file.Close() // Close the file in case of an error
		return
	}
	file.Close()

	// Start RPC
	listener, err := net.Listen("tcp", n.Addr)
	if err != nil {
		n.Print(fmt.Sprintf("Error starting RPC server: %v", err))
		return
	}
	defer listener.Close()

	rpcServer := rpc.NewServer()
	err = rpcServer.Register(n)
	if err != nil {
		n.Print(fmt.Sprintf("Error registering RPC server: %v", err))
	}
	rpcServer.Accept(listener)
}

type PingRequest struct{}

type PingResponse struct {
	Message string
	Name    string
}

func (n *Node) Ping(req *PingRequest, res *PingResponse) error {
	n.Print("Pinged")
	res.Message = "Pong from coordinator"
	res.Name = n.Type + "-" + n.Name
	if n.Type == "Coordinator" {
		res.Name = n.Type
	}
	return nil
}

type HealthCheckRequest struct{}

type HealthCheckResponse struct {
	Status string
}

func (n *Node) HealthCheck(req *HealthCheckRequest, res *HealthCheckResponse) error {
	res.Status = "OK"
	return nil
}

type GetInfoRequest struct{}

type GetInfoResponse struct {
	Name string
	Addr string
	Type string
}

func (n *Node) GetInfo(req *GetInfoRequest, res *GetInfoResponse) error {
	res.Name = n.Name
	res.Addr = n.Addr
	res.Type = n.Type
	return nil
}

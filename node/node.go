package node

import (
	"fmt"
	"net"
	"net/rpc"
)

type Node struct {
	Name string
	Addr string
	Type string
	// Coordinator Related
	c_participantClients map[string]*rpc.Client
	// Participant Related
	p_coordinatorClient *rpc.Client
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
	res.Name = "coordinator"
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

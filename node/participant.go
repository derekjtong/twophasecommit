package node

import (
	"fmt"
	"net/rpc"
)

type ParticipantConnectToCoordinatorRequest struct {
	Addr string
}
type ParticipantConnectToCoordinatorResponse struct{}

func (n *Node) ParticipantConnectToCoordinator(req *ParticipantConnectToCoordinatorRequest, res *ParticipantConnectToCoordinatorResponse) error {
	coordinatorClient, err := rpc.Dial("tcp", req.Addr)
	if err != nil {
		n.Print(fmt.Sprintf("Error connecting to coordinator: %v", err))
	}
	var addReq = AddParticipantRequest{Name: n.Name, Addr: n.Addr}
	var addRes AddParticipantResponse
	if err := coordinatorClient.Call("Node.AddParticipant", &addReq, &addRes); err != nil {
		n.Print(fmt.Sprintf("Error AddParticipant RPC: %v\n", err))
		return fmt.Errorf("error callingAddParticipant RPC: %v", err)
	}
	n.p_coordinatorClient = coordinatorClient

	n.Print(fmt.Sprintf("Connected to coordinator: %v", req.Addr))
	return nil
}

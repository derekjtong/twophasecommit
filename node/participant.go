package node

import (
	"fmt"
	"net/rpc"
)

type SetCoordinatorRequest struct {
	Addr string
}
type SetCoordinatorResponse struct{}

func (n *Node) SetCoordinator(req *SetCoordinatorRequest, res *SetCoordinatorResponse) error {
	coordinatorClient, err := rpc.Dial("tcp", req.Addr)
	if err != nil {
		n.Print(fmt.Sprintf("Error connecting to coordinator: %v", err))
	}
	n.p_coordinatorClient = coordinatorClient

	n.Print(fmt.Sprintf("Set coordinator: %v", req.Addr))
	return nil
}

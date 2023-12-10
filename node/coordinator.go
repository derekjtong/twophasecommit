package node

import (
	"fmt"
	"net/rpc"
)

type AddParticipantRequest struct {
	Name string
	Addr string
}

type AddParticipantResponse struct {
}

func (n *Node) AddParticipant(req *AddParticipantRequest, res *AddParticipantResponse) error {
	if n.Type != "Coordinator" {
		return fmt.Errorf("this node is not a coordinator")
	}
	if n.c_participantClients == nil {
		n.c_participantClients = make(map[string]*ConnectionData)
	}

	client, err := rpc.Dial("tcp", req.Addr)
	if err != nil {
		return err
	}
	var data = ConnectionData{
		Name:    req.Name,
		Address: req.Addr,
		Client:  client,
	}
	n.c_participantClients[req.Name] = &data
	n.Print(fmt.Sprintf("Added %v to participant list", req.Name))
	// TODO: use uuid for key, or check for repeat names
	return nil
}

type ListParticipantsRequest struct{}
type ListParticipantsResponse struct {
	Names     []string
	Addresses []string
}

func (n *Node) ListParticipants(req *ListParticipantsRequest, res *ListParticipantsResponse) error {
	if n.Type != "Coordinator" {
		n.Print("Requesting participant list from coordinator")
		if n.p_coordinatorClient == nil {
			return fmt.Errorf("coordinator client not set")
		}
		err := n.p_coordinatorClient.Call("Node.ListParticipants", &req, &res)
		if err != nil {
			return fmt.Errorf("failed to get participant list from coordinator: %v", err)
		}
	} else {
		n.Print("Listing participants as coordinator")

		res.Names = make([]string, 0, len(n.c_participantClients))
		res.Addresses = make([]string, 0, len(n.c_participantClients))

		for _, data := range n.c_participantClients {
			res.Names = append(res.Names, data.Name)
			res.Addresses = append(res.Addresses, data.Address)
		}
	}
	return nil
}

type ParticipantCoordinatorSendRequest struct {
	TargetAddr string
	TargetName string
	Amount     float64
	SenderAddr string
	SenderName string
}

type ParticipantCoordinatorSendResponse struct {
	TID string
}

func (n *Node) ParticipantCoordinatorSend(req *ParticipantCoordinatorSendRequest, res *ParticipantCoordinatorSendResponse) error {
	// Step 1: Prepare Phase
	n.Print("Prepare phase")
	okA, errA := n.sendPrepare(req.SenderName, req)
	okB, errB := n.sendPrepare(req.SenderName, req)

	// Check if both participants are ready and no errors occurred
	if errA != nil || errB != nil || !okA || !okB {
		// If not, send rollback message to both participants and return an error
		n.sendRollback("ParticipantA")
		n.sendRollback("ParticipantB")
		return fmt.Errorf("transaction aborted during prepare phase")
	}

	// Step 2: Commit Phase
	n.Print("Commit phase")
	n.sendCommit("ParticipantA")
	n.sendCommit("ParticipantB")

	return nil
}

func (n *Node) sendPrepare(name string, req *ParticipantCoordinatorSendRequest) (bool, error) {
	prepareReq := PrepareRequest{
		TargetAddr: req.TargetAddr,
		Amount:     req.Amount,
	}
	var prepareRes PrepareResponse
	client := n.c_participantClients[name].Client

	// Send Prepare
	err := client.Call("Node.Prepare", &prepareReq, &prepareRes)
	if err != nil {
		return false, err
	}
	return prepareRes.Ready, nil
}

func (n *Node) sendCommit(participant string) {
}

func (n *Node) sendRollback(participant string) {
}

package node

import "fmt"

// RPC: Client to Participant transaction request. Forwards request to Coordinator.
type ClientParticipantSendRequest struct {
	TargetAddr string
	TargetName string
	Amount     float64
}
type ClientParticipantSendResponse struct{}

func (n *Node) ClientParticipantSend(req *ParticipantCoordinatorSendRequest, res *ParticipantCoordinatorSendResponse) error {
	if n.Type != "Participant" {
		return fmt.Errorf("must be participant to send")
	}
	coordReq := ParticipantCoordinatorSendRequest{
		TargetAddr: req.TargetAddr,
		TargetName: req.TargetName,
		Amount:     req.Amount,
		SenderAddr: n.Addr,
		SenderName: n.Name,
	}
	var coordRes ParticipantCoordinatorSendResponse
	err := n.p_coordinatorClient.Call("Node.ParticipantCoordinatorSend", &coordReq, &coordRes)
	if err != nil {
		return fmt.Errorf("error initiating send with coordinator: %v", err)
	}
	fmt.Printf("Send to %v initiated for amount %v\n", req.TargetAddr, req.Amount)
	return nil
}

// RPC: Prepare requset
type PrepareRequest struct {
	TargetAddr string
	Amount     float64
}
type PrepareResponse struct {
	Ready bool
}

func (n *Node) Prepare(req *PrepareRequest, res *PrepareResponse) error {
	n.Print("receive prepare req\n")
	if n.promisedCommit {
		n.Print("not ready 1\n")
		res.Ready = false
		return nil
	}
	n.promisedCommit = true
	if n.canCommit(req) {
		n.Print("ready\n")
		res.Ready = true
		return nil
	} else {
		n.Print("not ready 2\n")
		n.promisedCommit = false
		res.Ready = false
	}
	return nil
}

// Check if participant can commit
func (n *Node) canCommit(req *PrepareRequest) bool {
	bal, err := n.getBalance()
	if err != nil {
		n.Print(fmt.Sprintf("Error getting balance: %v", err))
		return false
	}
	return bal+req.Amount >= 0
}

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
	n.Print("----Transaction Request----")
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
	n.Print(fmt.Sprintf("Sent %v to %v\n", req.Amount, req.TargetAddr))
	return nil
}

// RPC: Prepare requset
type ReceivePrepareRequest struct {
	TargetAddr string
	Amount     float64
}
type ReceivePrepareResponse struct {
	Ready bool
}

func (n *Node) ReceivePrepare(req *ReceivePrepareRequest, res *ReceivePrepareResponse) error {
	n.Print("Receive prepare")
	if n.promisedCommit {
		n.Print("Rejected, already promised")
		res.Ready = false
		return nil
	}
	n.promisedCommit = true
	bal, err := n.getBalance()
	if err != nil {
		n.Print(fmt.Sprintf("Error getting balance: %v", err))
	}

	if bal+req.Amount >= 0 {
		n.Print("Accepted")
		n.transactionAmount = req.Amount
		res.Ready = true
		return nil
	} else {
		n.Print("Rejected, balance too low")
		n.promisedCommit = false
		res.Ready = false
	}
	return nil
}

type ReceiveCommitRequest struct{}

type ReceiveCommitResponse struct{}

func (n *Node) ReceiveCommit(req *ReceiveCommitRequest, res *ReceiveCommitResponse) error {
	n.Print("Receive commit")
	bal, err := n.getBalance()
	if err != nil {
		fmt.Printf("Error getting balance: %v\n", err)
	}

	err = n.WriteBalance(bal + n.transactionAmount)
	if err != nil {
		fmt.Printf("Error writing balance: %v\n", err)
	}
	n.promisedCommit = false
	return nil
}

type ReceiveRollbackRequest struct{}

type ReceiveRollbackResponse struct{}

func (n *Node) ReceiveRollback(req *ReceiveRollbackRequest, res *ReceiveRollbackResponse) error {
	n.Print("Receive rollback")

	n.promisedCommit = false
	return nil
}

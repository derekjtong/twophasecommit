package node

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// RPC: Client to Participant transaction request. Forwards request to Coordinator.
type ClientParticipantSendRequest struct {
	TargetAddr string
	TargetName string
	Amount     float64
}
type ClientParticipantSendResponse struct{}

func (n *Node) ClientParticipantSend(req *ParticipantCoordinatorSendRequest, res *ParticipantCoordinatorSendResponse) error {
	n.Print("----Transaction Request Start----")
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
		return fmt.Errorf("coordinator error: %v", err)
	}
	n.Print(fmt.Sprintf("Sent %v to %v", req.Amount, req.TargetAddr))
	n.Print("----Transaction Request End----")
	return nil
}

// RPC: Prepare requset
type ReceivePrepareRequest struct {
	TransactionID uuid.UUID
	TargetAddr    string
	Amount        float64
}
type ReceivePrepareResponse struct {
	Response string
}

func (n *Node) ReceivePrepare(req *ReceivePrepareRequest, res *ReceivePrepareResponse) error {
	if n.promisedCommit {
		n.Print(fmt.Sprintf(colorRed + "Response: VoteAbort (already promised)" + colorReset))
		res.Response = "VoteAbort"
		n.LogTransaction("VoteAbort", req.TransactionID)
		return errors.New("already promised")
	}
	n.promisedCommit = true
	bal, err := n.getBalance()
	if err != nil {
		n.promisedCommit = false
		n.Print(fmt.Sprintf(colorRed + "Response: VoteAbort (error getting balance)" + colorReset))
		res.Response = "VoteAbort"
		n.LogTransaction("VoteAbort", req.TransactionID)
		return err
	}
	if bal+req.Amount >= 0 {
		n.Print(fmt.Sprintf(colorGreen + "Response: VoteCommit" + colorReset))
		n.transactionAmount = req.Amount
		res.Response = "VoteCommit"
		n.LogTransaction("VoteCommit", req.TransactionID)
		return nil
	}
	n.promisedCommit = false
	n.Print(fmt.Sprintf(colorRed + "Response: VoteAbort (insufficient balance)" + colorReset))
	res.Response = "VoteAbort"
	n.LogTransaction("VoteAbort", req.TransactionID)
	return errors.New("insufficient balance")
}

type ReceiveCommitRequest struct {
	TransactionID uuid.UUID
}

type ReceiveCommitResponse struct {
}

func (n *Node) ReceiveCommit(req *ReceiveCommitRequest, res *ReceiveCommitResponse) error {
	n.LogTransaction("COMMIT", req.TransactionID)
	n.Print(fmt.Sprintf(colorGreen + "Committing" + colorReset))
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

type ReceiveAbortRequest struct {
	TransactionID uuid.UUID
}

type ReceiveAbortResponse struct {
}

func (n *Node) ReceiveAbort(req *ReceiveAbortRequest, res *ReceiveAbortResponse) error {
	n.LogTransaction("ABORT", req.TransactionID)
	n.Print(fmt.Sprintf(colorRed + "Aborting" + colorReset))
	n.promisedCommit = false
	return nil
}

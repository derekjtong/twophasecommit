package node

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// RPC: Client to Participant transaction request. Forwards request to Coordinator.
type ClientParticipantTransactionRequest struct {
	Transactions []Transaction
}
type ClientParticipantTransactionResponse struct{}

func (n *Node) ClientParticipantTransaction(req *ClientParticipantTransactionRequest, res *ClientParticipantTransactionResponse) error {
	n.Print("----Transaction Request Start----")
	if n.Type != "Participant" {
		return fmt.Errorf("must be participant to send")
	}
	coordReq := ParticipantCoordinatorTransactionRequest{
		Transactions: req.Transactions,
	}
	var coordRes ParticipantCoordinatorTransactionResponse
	err := n.p_coordinatorClient.Call("Node.ParticipantCoordinatorTransaction", &coordReq, &coordRes)
	if err != nil {
		return fmt.Errorf("coordinator error: %v", err)
	}
	n.Print("----Transaction Request End----")
	return nil
}

// RPC: Prepare requset
type ReceivePrepareRequest struct {
	TransactionID uuid.UUID
	Amount        float64
	Operation     string
}
type ReceivePrepareResponse struct {
	Response string
}

func (n *Node) ReceivePrepare(req *ReceivePrepareRequest, res *ReceivePrepareResponse) error {
	if n.sleepBeforeRespondingToCoordinator {
		time.Sleep(10 * time.Second)
	}
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
		if n.sleepAfterRespondingToCoordinator {
			time.Sleep(10 * time.Second)
		}
		return err
	}
	var newBalance float64
	switch req.Operation {
	case "add":
		newBalance = bal + req.Amount
	case "multiply":
		newBalance = bal * req.Amount
	case "subtract":
		newBalance = bal - req.Amount
	}
	if newBalance >= 0 {
		n.Print(fmt.Sprintf(colorGreen + "Response: VoteCommit" + colorReset))
		n.transactionNewBalance = newBalance
		res.Response = "VoteCommit"
		n.LogTransaction("VoteCommit", req.TransactionID)
		if n.sleepAfterRespondingToCoordinator {
			time.Sleep(10 * time.Second)
		}
		return nil
	}
	n.promisedCommit = false
	n.Print(fmt.Sprintf(colorRed + "Response: VoteAbort (insufficient balance)" + colorReset))
	res.Response = "VoteAbort"
	n.LogTransaction("VoteAbort", req.TransactionID)
	if n.sleepAfterRespondingToCoordinator {
		time.Sleep(10 * time.Second)
	}
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

	err := n.WriteBalance(n.transactionNewBalance)
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

// TODO: Client rpc call
type SimulateDelayRequest struct {
	sleepBeforeRespondingToCoordinator bool
	sleepAfterRespondingToCoordinator  bool
}

type SimulateDelayResponse struct{}

func (n *Node) SimulateDelay(req *SimulateDelayRequest, res *SimulateDelayResponse) error {
	n.sleepBeforeRespondingToCoordinator = req.sleepBeforeRespondingToCoordinator
	n.sleepAfterRespondingToCoordinator = req.sleepAfterRespondingToCoordinator
	return nil
}

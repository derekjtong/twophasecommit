package node

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// RPC: Participant to Coordinator transaction request
type ParticipantCoordinatorTransactionRequest struct {
	TargetAddr      string
	TargetName      string
	TargetOperation string
	TargetAmount    float64
	SenderAddr      string
	SenderName      string
	SenderOperation string
	SenderAmount    float64
}

type ParticipantCoordinatorTransactionResponse struct{}

func (n *Node) ParticipantCoordinatorTransaction(req *ParticipantCoordinatorTransactionRequest, res *ParticipantCoordinatorTransactionResponse) error {
	// Generate Transaction ID
	transactionID := uuid.New()
	n.Print(fmt.Sprintf("---Transaction ID: %s---", transactionID))

	n.LogTransaction("PREPARE", transactionID)

	// Step 1: Prepare Phase
	n.Print("---Prepare phase---")
	errA := n.sendPrepare(req.SenderName, req.SenderAmount, req.SenderOperation, transactionID)
	errB := n.sendPrepare(req.TargetName, req.TargetAmount, req.TargetOperation, transactionID)

	// Check if both participants are ready and no errors occurred
	if errA != nil || errB != nil {
		n.LogTransaction("ABORT", transactionID)

		// Send rollback messages
		n.sendRollback(req.SenderName, transactionID)
		n.sendRollback(req.TargetName, transactionID)

		var errMsg string
		if errA != nil {
			errMsg += fmt.Sprintf("participant-%s: %v. ", req.SenderName, errA)
		}
		if errB != nil {
			errMsg += fmt.Sprintf("participant-%s: %v. ", req.TargetName, errB)
		}

		return fmt.Errorf("transaction aborted. %s", errMsg)
	}
	n.LogTransaction("COMMIT", transactionID)

	// Step 2: Commit Phase
	n.Print("---Commit phase---")
	n.sendCommit(req.SenderName, transactionID)
	n.sendCommit(req.TargetName, transactionID)

	return nil
}

// Send prepare request
func (n *Node) sendPrepare(name string, amount float64, operation string, transactionID uuid.UUID) error {
	n.Print("Request: CanCommit?")
	req := ReceivePrepareRequest{
		TransactionID: transactionID,
		Amount:        amount,
		Operation:     operation,
	}
	var res ReceivePrepareResponse
	client := n.c_participantClients[name].Client

	// Send Prepare
	err := client.Call("Node.ReceivePrepare", &req, &res)
	if err != nil {
		return err
	}
	if res.Response == "VoteAbort" {
		return nil
	} else if res.Response == "VoteCommit" {
		return nil
	}
	return errors.New("received invalid response")
}

// Send commit
func (n *Node) sendCommit(name string, transactionID uuid.UUID) {
	n.Print("Request: DoCommit")
	client := n.c_participantClients[name].Client
	var req ReceiveCommitRequest = ReceiveCommitRequest{
		TransactionID: transactionID,
	}
	var res ReceiveCommitResponse
	err := client.Call("Node.ReceiveCommit", &req, &res)
	if err != nil {
		n.Print(fmt.Sprintf("Error sending commit: %v", err))
	}
}

func (n *Node) sendRollback(name string, transactionID uuid.UUID) {
	n.Print("Request: DoAbort")
	client := n.c_participantClients[name].Client
	var req ReceiveAbortRequest = ReceiveAbortRequest{
		TransactionID: transactionID,
	}
	var res ReceiveCommitResponse
	err := client.Call("Node.ReceiveAbort", &req, &res)
	if err != nil {
		n.Print(fmt.Sprintf("Error rolling back: %v", err))
	}
}

package node

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Transaction struct {
	Addr      string
	Name      string
	Operation string
	Amount    float64
}

// RPC: Participant to Coordinator transaction request
type ParticipantCoordinatorTransactionRequest struct {
	Transactions []Transaction
}

type ParticipantCoordinatorTransactionResponse struct{}

func (n *Node) ParticipantCoordinatorTransaction(req *ParticipantCoordinatorTransactionRequest, res *ParticipantCoordinatorTransactionResponse) error {
	// Generate Transaction ID
	transactionID := uuid.New()
	n.Print(fmt.Sprintf("---Transaction ID: %s---", transactionID))

	n.LogTransaction("PREPARE", transactionID)
	// Step 1: Prepare Phase
	n.Print("---Prepare phase---")
	var combinedError string
	for _, tx := range req.Transactions {
		err := n.sendPrepare(tx.Name, tx.Amount, tx.Operation, transactionID, req.Transactions)
		if err != nil {
			if combinedError != "" {
				combinedError += "; "
			}
			combinedError += fmt.Sprintf("transaction aborted for %s: %v", tx.Name, err)
		}
	}
	if combinedError != "" {
		n.LogTransaction("ABORT", transactionID)
		for _, txRollback := range req.Transactions {
			n.sendAbort(txRollback.Name, transactionID)
		}
		return errors.New(combinedError)
	}

	n.LogTransaction("COMMIT", transactionID)
	// Step 2: Commit Phase
	n.Print("---Commit phase---")
	for _, tx := range req.Transactions {
		n.sendCommit(tx.Name, transactionID)
	}

	return nil
}

// Send Prepare/CanCommit? request
func (n *Node) sendPrepare(name string, amount float64, operation string, transactionID uuid.UUID, transactions []Transaction) error {
	n.Print("Request: CanCommit?")

	req := ReceivePrepareRequest{
		Transactions:  transactions,
		TransactionID: transactionID,
		Amount:        amount,
		Operation:     operation,
	}
	var res ReceivePrepareResponse

	// Channel to communicate the outcome of the RPC call
	done := make(chan error, 1)

	// Perform the RPC call in a goroutine
	go func() {
		err := n.c_participantClients[name].Client.Call("Node.ReceivePrepare", &req, &res)
		done <- err
	}()

	// Implementing timeout using select
	select {
	case err := <-done:
		if err != nil {
			return err
		}
		if res.Response == "VoteAbort" {
			return errors.New("vote aborted by participant")
		} else if res.Response != "VoteCommit" {
			return errors.New("received invalid response")
		}
		return nil
	case <-time.After(5 * time.Second):
		return errors.New("transaction aborted due to timeout")
	}
}

// Send DoCommit
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

// Send DoAbort
func (n *Node) sendAbort(name string, transactionID uuid.UUID) {
	n.Print("Request: DoAbort")
	client := n.c_participantClients[name].Client
	var req ReceiveAbortRequest = ReceiveAbortRequest{
		TransactionID: transactionID,
	}
	var res ReceiveCommitResponse
	err := client.Call("Node.ReceiveAbort", &req, &res)
	if err != nil {
		n.Print(fmt.Sprintf("Error aborting back: %v", err))
	}
}

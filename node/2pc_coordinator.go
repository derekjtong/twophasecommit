package node

import (
	"errors"
	"fmt"

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
	for _, tx := range req.Transactions {
		err := n.sendPrepare(tx.Name, tx.Amount, tx.Operation, transactionID)
		if err != nil {
			n.LogTransaction("ABORT", transactionID)
			for _, txRollback := range req.Transactions {
				n.sendRollback(txRollback.Name, transactionID)
			}
			return fmt.Errorf("transaction aborted for %s: %v", tx.Name, err)
		}
	}
	n.LogTransaction("COMMIT", transactionID)

	// Step 2: Commit Phase
	n.Print("---Commit phase---")
	for _, tx := range req.Transactions {
		n.sendCommit(tx.Name, transactionID)
	}

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

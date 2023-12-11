package node

import (
	"bufio"
	"errors"
	"fmt"
	"net/rpc"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// RPC: Client to Participant transaction request. Forward request to Coordinator.
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

// RPC: Process received Prepare/CanCommit? request
type ReceivePrepareRequest struct {
	Transactions  []Transaction
	TransactionID uuid.UUID
	Amount        float64
	Operation     string
}
type ReceivePrepareResponse struct {
	Response string
}

func (n *Node) ReceivePrepare(req *ReceivePrepareRequest, res *ReceivePrepareResponse) error {
	if n.sleepBeforeRespondingToCoordinator {
		n.rejectIncoming = true
		n.Print("Simulating delay (before)")
		time.Sleep(10 * time.Second)
		n.rejectIncoming = false
	}
	n.LogTransaction("Prepare", req.TransactionID)

	// Mutex to protect n.promisedCommit bool
	n.commitMutex.Lock()
	defer n.commitMutex.Unlock()
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
			go func() {
				n.Print("Simulating delay (after)")
				n.rejectIncoming = true
				time.Sleep(10 * time.Second)
				n.rejectIncoming = false
			}()
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

		// Monitor log file to check for transaction completion

		n.stopMonitoring = make(chan bool)
		go n.monitorTransactionStatus(req.TransactionID, req.Transactions)

		if n.sleepAfterRespondingToCoordinator {
			go func() {
				n.Print("Simulating delay (after)")
				n.rejectIncoming = true
				time.Sleep(10 * time.Second)
				n.rejectIncoming = false
			}()
		}
		return nil
	}
	n.promisedCommit = false
	n.Print(fmt.Sprintf(colorRed + "Response: VoteAbort (insufficient balance)" + colorReset))
	res.Response = "VoteAbort"
	n.LogTransaction("VoteAbort", req.TransactionID)
	if n.sleepAfterRespondingToCoordinator {
		go func() {
			n.Print("Simulating delay (after)")
			n.rejectIncoming = true
			time.Sleep(10 * time.Second)
			n.rejectIncoming = false
		}()
	}
	return errors.New("insufficient balance")
}

// RPC: Process received DoCommit request
type ReceiveCommitRequest struct {
	TransactionID uuid.UUID
}

type ReceiveCommitResponse struct {
}

func (n *Node) ReceiveCommit(req *ReceiveCommitRequest, res *ReceiveCommitResponse) error {
	if n.rejectIncoming {
		return fmt.Errorf("rejected, simulating crash")
	}
	n.commitMutex.Lock()
	defer n.commitMutex.Unlock()
	n.LogTransaction("COMMIT", req.TransactionID)
	n.Print(fmt.Sprintf(colorGreen + "Committing" + colorReset))

	err := n.WriteBalance(n.transactionNewBalance)
	if err != nil {
		fmt.Printf("Error writing balance: %v\n", err)
	}
	n.promisedCommit = false
	return nil
}

// RPC: Process received DoAbort request
type ReceiveAbortRequest struct {
	TransactionID uuid.UUID
}

type ReceiveAbortResponse struct {
}

func (n *Node) ReceiveAbort(req *ReceiveAbortRequest, res *ReceiveAbortResponse) error {
	if n.rejectIncoming {
		return fmt.Errorf("rejected, simulating crash")
	}
	n.commitMutex.Lock()
	defer n.commitMutex.Unlock()
	n.LogTransaction("ABORT", req.TransactionID)
	n.Print(fmt.Sprintf(colorRed + "Aborting" + colorReset))
	n.promisedCommit = false
	n.stopMonitoring <- true
	return nil
}

// TODO: Client rpc call
type SimulateDelayRequest struct {
	SleepBefore bool
	SleepAfter  bool
}

type SimulateDelayResponse struct{}

func (n *Node) SimulateDelay(req *SimulateDelayRequest, res *SimulateDelayResponse) error {
	if req.SleepBefore && req.SleepAfter {
		return fmt.Errorf("only one of SleepBefore or SleepAfter can be true")
	}

	n.sleepBeforeRespondingToCoordinator = req.SleepBefore
	n.sleepAfterRespondingToCoordinator = req.SleepAfter

	return nil
}

// RPC: Other participant doesn't have info, querying this node
type P2PQueryTransactionStatusRequest struct {
	RequesterAddr string
	TransactionID uuid.UUID
}
type P2PQueryTranactionStatusResponse struct {
}

type LogEntry struct {
	TransactionID uuid.UUID
	Status        string
}

func (n *Node) P2PQueryTransactionStatus(req *P2PQueryTransactionStatusRequest, res *P2PQueryTranactionStatusResponse) error {
	// Use the checkLocalLogForStatus function to get the transaction status
	status, found := n.checkLocalLogForStatus(req.TransactionID)
	if !found {
		return fmt.Errorf("transaction status not found for %s", req.TransactionID)
	}

	n.Print(fmt.Sprintf("Transaction %s status: %s", req.TransactionID, status))
	client, err := rpc.Dial("tcp", req.RequesterAddr)
	if err != nil {
		fmt.Printf("Error starting p2p rpc client: %v", err)
		return fmt.Errorf("error starting p2p rpc client: %v", err)
	}
	if status == "ABORT" {
		var req ReceiveAbortRequest = ReceiveAbortRequest{
			TransactionID: req.TransactionID,
		}
		var res ReceiveCommitResponse
		err := client.Call("Node.ReceiveAbort", &req, &res)
		if err != nil {
			return fmt.Errorf("error sending abort: %v", err)
		}
	} else if status == "COMMIT" {
		var req ReceiveCommitRequest = ReceiveCommitRequest{
			TransactionID: req.TransactionID,
		}
		var res ReceiveCommitResponse
		err := client.Call("Node.ReceiveCommit", &req, &res)
		if err != nil {
			n.Print(fmt.Sprintf("Error sending commit: %v", err))
		}
	} else {
		return fmt.Errorf("invalid status")
	}
	return nil
}

func (n *Node) monitorTransactionStatus(transactionID uuid.UUID, transactions []Transaction) {
	n.Print("Starting thread to monitor transaction status")
	for {
		if n.rejectIncoming {
			time.Sleep(1 * time.Second)
			n.Print("monitorTransactionStatus paused to simulate crash...")
			continue
		}
		// foundStatus := false
		time.Sleep(500 * time.Millisecond) // Delay between queries
		select {
		case <-n.stopMonitoring:
			n.Print("Monitoring stopped")
			return
		default:
			for _, transaction := range transactions {
				// Don't query self
				if transaction.Addr == n.Addr {
					continue
				}

				// First, check the local log file
				if status, found := n.checkLocalLogForStatus(transactionID); found {
					n.Print(fmt.Sprintf("(check) Transaction %s status in local log: %s", transactionID, status))
					return // Exit the function if status is found
				}

				// Query other participants if not found in local log
				req := P2PQueryTransactionStatusRequest{TransactionID: transactionID, RequesterAddr: n.Addr}
				var res P2PQueryTranactionStatusResponse

				n.Print(fmt.Sprintf("Requesting info from participant %s", transaction.Name))
				client, err := rpc.Dial("tcp", transaction.Addr)
				if err != nil {
					n.Print(fmt.Sprintf("Error dialing %s: %v", transaction.Name, err))
					continue // Try the next participant
				}

				err = client.Call("Node.P2PQueryTransactionStatus", &req, &res)
				client.Close() // Close the client after the call
				if err != nil {
					n.Print(fmt.Sprintf("Error querying transaction status from %s: %v", transaction.Name, err))
					continue // Try the next participant
				}

				// Check if status has been updated in the local log
				if updatedStatus, updated := n.checkLocalLogForStatus(transactionID); updated {
					n.Print(fmt.Sprintf("Updated transaction %s status in local log: %s", transactionID, updatedStatus))
					// foundStatus = true
					break // Break out of the inner loop if status is updated
				}

				time.Sleep(5 * time.Second) // Delay between queries
			}

			// Check if the status was found in this iteration
			// if foundStatus {
			// 	break // Break out of the infinite loop
			// }
		}
	}
}

func (n *Node) checkLocalLogForStatus(transactionID uuid.UUID) (string, bool) {
	logFile := filepath.Join("node_log", fmt.Sprintf("%s-%s.log", n.Type, n.Name))
	file, err := os.Open(logFile)
	if err != nil {
		n.Print(fmt.Sprintf("Error opening log file: %v", err))
		return "", false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		var tidStr string
		var status string
		_, err := fmt.Sscanf(line, "%s - %s", &tidStr, &status)
		if err != nil {
			continue
		}
		tid, err := uuid.Parse(tidStr)
		if err != nil {
			fmt.Printf("Error parsing UUID: %v\n", err)
			continue
		}
		if tid == transactionID {
			if status == "COMMIT" || status == "ABORT" {
				return status, true
			}
		}
	}

	if err := scanner.Err(); err != nil {
		n.Print(fmt.Sprintf("Error reading log file: %v", err))
	}

	return "", false
}

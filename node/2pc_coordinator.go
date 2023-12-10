package node

import "fmt"

// RPC: Participant to Coordinator transaction request
type ParticipantCoordinatorSendRequest struct {
	TargetAddr string
	TargetName string
	Amount     float64
	SenderAddr string
	SenderName string
}
type ParticipantCoordinatorSendResponse struct{}

func (n *Node) ParticipantCoordinatorSend(req *ParticipantCoordinatorSendRequest, res *ParticipantCoordinatorSendResponse) error {
	// Step 1: Prepare Phase
	n.Print("Prepare phase")
	okA, errA := n.sendPrepare(req.SenderName, -req.Amount)
	okB, errB := n.sendPrepare(req.TargetName, req.Amount)

	// Check if both participants are ready and no errors occurred
	if errA != nil || errB != nil || !okA || !okB {
		// Send rollback messages
		n.sendRollback(req.SenderName)
		n.sendRollback(req.TargetName)

		var errMsg string
		if errA != nil {
			errMsg += fmt.Sprintf("Error during prepare phase for %s: %v. ", req.SenderName, errA)
		}
		if errB != nil {
			errMsg += fmt.Sprintf("Error during prepare phase for %s: %v. ", req.TargetName, errB)
		}
		if !okA {
			errMsg += fmt.Sprintf("participant %s was not ready to commit. ", req.SenderName)
		}
		if !okB {
			errMsg += fmt.Sprintf("participant %s was not ready to commit. ", req.TargetName)
		}

		return fmt.Errorf("transaction aborted. %s rollback initiated", errMsg)
	}

	// Step 2: Commit Phase
	n.Print("Commit phase")
	n.sendCommit(req.SenderName)
	n.sendCommit(req.TargetName)

	return nil
}

// Send prepare request
func (n *Node) sendPrepare(name string, amount float64) (bool, error) {
	req := ReceivePrepareRequest{
		Amount: amount,
	}
	var res ReceivePrepareResponse
	client := n.c_participantClients[name].Client

	// Send Prepare
	err := client.Call("Node.ReceivePrepare", &req, &res)
	if err != nil {
		return false, err
	}
	return res.Ready, nil
}

// Send commit
func (n *Node) sendCommit(name string) {
	client := n.c_participantClients[name].Client
	var req ReceiveCommitRequest
	var res ReceiveCommitResponse
	err := client.Call("Node.ReceiveCommit", &req, &res)
	if err != nil {
		n.Print(fmt.Sprintf("Error sending commit: %v", err))
	}
}

func (n *Node) sendRollback(name string) {
	client := n.c_participantClients[name].Client
	var req ReceiveRollbackRequest
	var res ReceiveCommitResponse
	err := client.Call("Node.ReceiveRollback", &req, &res)
	if err != nil {
		n.Print(fmt.Sprintf("Error rolling back: %v", err))
	}
}

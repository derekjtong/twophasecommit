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
			errMsg += fmt.Sprintf("%s was not ready to commit. ", req.SenderName)
		}
		if !okB {
			errMsg += fmt.Sprintf("%s was not ready to commit. ", req.TargetName)
		}

		return fmt.Errorf("transaction aborted. Details: %s", errMsg)
	}

	// Step 2: Commit Phase
	n.Print("Commit phase")
	n.sendCommit("ParticipantA")
	n.sendCommit("ParticipantB")

	return nil
}

// Send prepare request
func (n *Node) sendPrepare(name string, amount float64) (bool, error) {
	prepareReq := PrepareRequest{
		Amount: amount,
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

// Send commit
func (n *Node) sendCommit(participant string) {
}

func (n *Node) sendRollback(participant string) {
}

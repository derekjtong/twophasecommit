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
		return fmt.Errorf("error initiating send with coordinator: %v", err)
	}
	n.Print(fmt.Sprintf("Sent %v to %v", req.Amount, req.TargetAddr))
	n.Print("----Transaction Request End----")
	return nil
}

// RPC: Prepare requset
type ReceivePrepareRequest struct {
	TargetAddr string
	Amount     float64
}
type ReceivePrepareResponse struct {
	Response string
}

func (n *Node) ReceivePrepare(req *ReceivePrepareRequest, res *ReceivePrepareResponse) error {
	if n.promisedCommit {
		n.Print(fmt.Sprintf(colorRed + "Response: VoteAbort (already promised)" + colorReset))
		res.Response = "VoteAbort"
		return nil
	}
	n.promisedCommit = true
	bal, err := n.getBalance()
	if err != nil {
		n.Print(fmt.Sprintf(colorRed + "Response: VoteAbort (error getting balance)" + colorReset))
		res.Response = "VoteAbort"
		n.promisedCommit = false
	}
	if bal+req.Amount >= 0 {
		n.Print(fmt.Sprintf(colorGreen + "Response: VoteCommit" + colorReset))
		n.transactionAmount = req.Amount
		res.Response = "VoteCommit"
		return nil
	}
	n.Print(fmt.Sprintf(colorRed + "Response: VoteAbort (unsufficient balance)" + colorReset))
	n.promisedCommit = false
	res.Response = "VoteAbort"
	return nil
}

type ReceiveCommitRequest struct{}

type ReceiveCommitResponse struct{}

func (n *Node) ReceiveCommit(req *ReceiveCommitRequest, res *ReceiveCommitResponse) error {
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

type ReceiveAbortRequest struct{}

type ReceiveAbortResponse struct{}

func (n *Node) ReceiveAbort(req *ReceiveAbortRequest, res *ReceiveAbortResponse) error {
	n.Print(fmt.Sprintf(colorRed + "Aborting" + colorReset))
	n.promisedCommit = false
	return nil
}

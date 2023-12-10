package node

import (
	"fmt"
	"net/rpc"
)

type AddParticipantRequest struct {
	Name string
	Addr string
}

type AddParticipantResponse struct {
}

func (n *Node) AddParticipant(req *AddParticipantRequest, res *AddParticipantResponse) error {
	if n.Type != "Coordinator" {
		return fmt.Errorf("this node is not a coordinator")
	}
	if n.c_participantClients == nil {
		n.c_participantClients = make(map[string]*rpc.Client)
	}

	client, err := rpc.Dial("tcp", req.Addr)
	if err != nil {
		return err
	}
	n.c_participantClients[req.Name] = client
	// TODO: use uuid for key, or check for repeat names
	return nil
}

type ListParticipantsRequest struct{}
type ListParticipantsResponse struct {
	Participants []string
}

func (n *Node) ListParticipants(req *ListParticipantsRequest, res *ListParticipantsResponse) error {
	if n.Type != "Coordinator" {
		return fmt.Errorf("this node is not a coordinator")
	}

	n.Print("Listing participants")

	// Initialize response slice
	res.Participants = make([]string, 0, len(n.c_participantClients))

	// Iterate over the map and collect participant names
	for name := range n.c_participantClients {
		res.Participants = append(res.Participants, name)
	}

	return nil
}

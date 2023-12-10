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
		n.c_participantClients = make(map[string]*ConnectionData)
	}

	client, err := rpc.Dial("tcp", req.Addr)
	if err != nil {
		return err
	}
	var data = ConnectionData{
		Name:    req.Name,
		Address: req.Addr,
		Client:  client,
	}
	n.c_participantClients[req.Name] = &data
	n.Print(fmt.Sprintf("Added %v to participant list", req.Name))
	// TODO: use uuid for key, or check for repeat names
	return nil
}

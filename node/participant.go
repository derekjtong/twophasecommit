package node

import (
	"fmt"
	"net/rpc"
	"os"
	"path/filepath"
	"strconv"
)

type ParticipantConnectToCoordinatorRequest struct {
	Addr string
}
type ParticipantConnectToCoordinatorResponse struct{}

func (n *Node) ParticipantConnectToCoordinator(req *ParticipantConnectToCoordinatorRequest, res *ParticipantConnectToCoordinatorResponse) error {
	coordinatorClient, err := rpc.Dial("tcp", req.Addr)
	if err != nil {
		n.Print(fmt.Sprintf("Error connecting to coordinator: %v", err))
	}
	var addReq = AddParticipantRequest{Name: n.Name, Addr: n.Addr}
	var addRes AddParticipantResponse
	if err := coordinatorClient.Call("Node.AddParticipant", &addReq, &addRes); err != nil {
		n.Print(fmt.Sprintf("Error AddParticipant RPC: %v\n", err))
		return fmt.Errorf("error callingAddParticipant RPC: %v", err)
	}
	n.p_coordinatorClient = coordinatorClient

	n.Print(fmt.Sprintf("Connected to coordinator: %v", req.Addr))
	return nil
}

type ParticipantInitiateTransferRequest struct {
	TargetAddr string
	Amount     float64
}
type ParticipantInitiateTransferResponse struct{}

func (n *Node) StartTransfer(req *ParticipantInitiateTransferRequest, res *ParticipantInitiateTransferResponse) error {
	if n.Type != "Participant" {
		return fmt.Errorf("must be participant to transfer")
	}
	coordReq := InitiateTransferRequest{
		TargetAddr: req.TargetAddr,
		Amount:     req.Amount,
	}
	var coordRes InitiateTransferResponse
	err := n.p_coordinatorClient.Call("Node.InitiateTransfer", &coordReq, &coordRes)
	if err != nil {
		return fmt.Errorf("error initiating transfer with coordinator: %v", err)
	}
	fmt.Printf("Transfer to %v initiated for amount %v\n", req.TargetAddr, req.Amount)
	return nil
}

type CheckFundsRequest struct {
	AccountAddr string
}

type CheckFundsResponse struct {
	Balance float64
	Success bool
}

func (n *Node) CheckFunds(req *CheckFundsRequest, res *CheckFundsResponse) error {
	// Retrieve the balance for the specified account
	balance, err := n.getBalance()
	if err != nil {
		// In case of an error (e.g., account not found)
		res.Success = false
		return fmt.Errorf("error retrieving balance: %v", err)
	}

	// Set the retrieved balance in the response
	res.Balance = balance
	res.Success = true
	return nil
}

func (n *Node) getBalance() (float64, error) {
	// Check if the data file exists
	nodeDataDir := "node_data"
	filename := fmt.Sprintf("%s-%s.data", n.Type, n.Name)
	filepath := filepath.Join(nodeDataDir, filename)

	_, statErr := os.Stat(filepath)
	if os.IsNotExist(statErr) {
		return 0.0, fmt.Errorf("data file not found")
	}

	// Open the data file
	file, openErr := os.Open(filepath)
	if openErr != nil {
		n.Print(fmt.Sprintf("Error opening data file: %v", openErr))
		return 0.0, openErr
	}
	defer file.Close()

	// Read the content from the file
	var content string
	_, readErr := fmt.Fscanf(file, "%s", &content)
	if readErr != nil {
		n.Print(fmt.Sprintf("Error reading data from file: %v", readErr))
		return 0.0, readErr
	}

	// Parse the content as a decimal number (float64)
	balance, parseErr := strconv.ParseFloat(content, 64)
	if parseErr != nil {
		n.Print(fmt.Sprintf("Error parsing balance from file: %v", parseErr))
		return 0.0, parseErr
	}

	return balance, nil
}

type DepositRequest struct {
	Amount float64
}

type DepositResponse struct {
	Success bool
	Message string
}

func (n *Node) Deposit(req *DepositRequest, res *DepositResponse) error {
	// Check if the deposit amount is valid (non-negative)
	if req.Amount < 0 {
		res.Success = false
		res.Message = "Invalid deposit amount (negative)"
		return nil
	}

	// TODO: Read/write lock
	n.fileLock.Lock()
	defer n.fileLock.Unlock()
	// Read the current balance from the data file
	currentBalance, err := n.getBalance()
	if err != nil {
		res.Success = false
		res.Message = fmt.Sprintf("Error reading current balance: %v", err)
		return err
	}
	// Calculate the new balance after deposit
	newBalance := currentBalance + req.Amount

	// Write the new balance to the data file
	writeErr := n.WriteBalance(newBalance)
	if writeErr != nil {
		res.Success = false
		res.Message = fmt.Sprintf("Error updating balance: %v", writeErr)
		return writeErr
	}

	res.Success = true
	res.Message = fmt.Sprintf("Deposit of %.2f successful. New balance: %.2f", req.Amount, newBalance)
	return nil
}

func (n *Node) WriteBalance(balance float64) error {
	// Check and create node_data directory
	nodeDataDir := "node_data"
	if _, err := os.Stat(nodeDataDir); os.IsNotExist(err) {
		err := os.Mkdir(nodeDataDir, 0755)
		if err != nil {
			n.Print(fmt.Sprintf("Error creating data directory: %v", err))
			return err
		}
	}

	// Create a data file for node
	filename := fmt.Sprintf("%s-%s.data", n.Type, n.Name)
	filepath := filepath.Join(nodeDataDir, filename)
	file, err := os.Create(filepath)
	if err != nil {
		n.Print(fmt.Sprintf("Error creating data file: %v", err))
		return err
	}

	// Write the balance as a decimal number to the file
	_, writeErr := fmt.Fprintf(file, "%.2f", balance)
	if writeErr != nil {
		n.Print(fmt.Sprintf("Error writing balance to file: %v", writeErr))
		file.Close()
		return writeErr
	}

	// Close file
	closeErr := file.Close()
	if closeErr != nil {
		n.Print(fmt.Sprintf("Error closing data file: %v", closeErr))
		return closeErr
	}

	return nil
}

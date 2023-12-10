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
	n.Print("Connected to coordinator")
	var addReq = AddParticipantRequest{Name: n.Name, Addr: n.Addr}
	var addRes AddParticipantResponse
	if err := coordinatorClient.Call("Node.AddParticipant", &addReq, &addRes); err != nil {
		n.Print(fmt.Sprintf("Error AddParticipant RPC: %v\n", err))
		return fmt.Errorf("error callingAddParticipant RPC: %v", err)
	}
	n.p_coordinatorClient = coordinatorClient

	return nil
}

type GetBalanceRequest struct {
	AccountAddr string
}

type GetBalanceResponse struct {
	Balance float64
}

func (n *Node) GetBalance(req *GetBalanceRequest, res *GetBalanceResponse) error {
	balance, err := n.getBalance()
	if err != nil {
		return fmt.Errorf("error retrieving balance: %v", err)
	}
	res.Balance = balance
	return nil
}

// TODO: Replace with mini-cloud
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
}

// TODO: Replace with mini-cloud
func (n *Node) Deposit(req *DepositRequest, res *DepositResponse) error {
	if req.Amount < 0 {
		return fmt.Errorf("cannot deposit negative")
	}

	// TODO: Read/write lock
	currentBalance, err := n.getBalance()
	if err != nil {
		n.Print(fmt.Sprintf("Error reading current balance: %v", err))
		return err
	}
	newBalance := currentBalance + req.Amount

	err = n.WriteBalance(newBalance)
	if err != nil {
		n.Print(fmt.Sprintf("Error updating balance: %v", err))
		return err
	}

	n.Print(fmt.Sprintf("Deposit %.2f successful. New balance: %.2f", req.Amount, newBalance))
	return nil
}

type WithdrawRequest struct {
	Amount float64
}

type WithdrawResponse struct {
}

// TODO: Replace with mini-cloud
func (n *Node) Withdraw(req *WithdrawRequest, res *WithdrawResponse) error {
	bal, err := n.getBalance()
	if err != nil {
		return err
	}
	if bal-req.Amount < 0 {
		return fmt.Errorf("insufficient funds")
	}

	// TODO: Read/write lock
	currentBalance, err := n.getBalance()
	if err != nil {
		n.Print(fmt.Sprintf("Error reading current balance: %v", err))
		return err
	}

	newBalance := currentBalance - req.Amount

	err = n.WriteBalance(newBalance)
	if err != nil {
		n.Print(fmt.Sprintf("Error updating balance: %v", err))
		return err
	}

	n.Print(fmt.Sprintf("Deposit %.2f successful. New balance: %.2f", req.Amount, newBalance))
	return nil
}

// TODO: Replace with mini-cloud
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

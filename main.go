package main

import (
	"bufio"
	"fmt"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"twophasecommit/node"
	"twophasecommit/utils"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go [server|client]")
		os.Exit(1)
	}

	if os.Args[1] == "server" {
		startServer()
	} else if os.Args[1] == "client" {
		startClient()
	} else {
		fmt.Println("Usage: go run main.go [server|client]")
		os.Exit(1)
	}
}

func startServer() {
	// Start Coordinator
	err := utils.ClearNodeDataDir()
	if err != nil {
		fmt.Printf("Error clearing node_data directory: %v\n", err)
		return
	}
	port, err := utils.FindAvailablePort()
	if err != nil {
		fmt.Printf("Error finding available port: %v\n", err)
		return
	}
	addrCoordinator := fmt.Sprintf("%s:%d", utils.IPAddress, port)
	go func(addr string) {
		coordinator, err := node.NewCoordinator(addr)
		if err != nil {
			fmt.Printf("Error creating coordinator: %v\n", err)
			return
		}
		coordinator.Start()
	}(addrCoordinator)
	err = utils.WaitForServerReady(addrCoordinator)
	if err != nil {
		fmt.Printf("Error waiting for C to be ready: %v\n", err)
		return
	}

	// Start Participant A
	port, err = utils.FindAvailablePort()
	if err != nil {
		fmt.Printf("Error finding available port: %v\n", err)
		return
	}
	addrParticipantA := fmt.Sprintf("%s:%d", utils.IPAddress, port)
	go func(addr string) {
		participantA, err := node.NewParticipant(addr, "A")
		if err != nil {
			fmt.Printf("Error creating participant A: %v\n", err)
			return
		}
		participantA.Start()
	}(addrParticipantA)
	err = utils.WaitForServerReady(addrParticipantA)
	if err != nil {
		fmt.Printf("Error waiting for P-A to be ready: %v\n", err)
		return
	}

	// Start Participant B
	port, err = utils.FindAvailablePort()
	if err != nil {
		fmt.Printf("Error finding available port: %v\n", err)
		return
	}
	addrParticipantB := fmt.Sprintf("%s:%d", utils.IPAddress, port)
	go func(addr string) {
		participantB, err := node.NewParticipant(addr, "B")
		if err != nil {
			fmt.Printf("Error creating participant A: %v\n", err)
			return
		}
		participantB.Start()
	}(addrParticipantB)
	err = utils.WaitForServerReady(addrParticipantB)
	if err != nil {
		fmt.Printf("Error waiting for P-B to be ready: %v\n", err)
		return
	}
	// Send coordinator to participants (prepare variables)
	var req = node.ParticipantConnectToCoordinatorRequest{Addr: addrCoordinator}
	var res node.ParticipantConnectToCoordinatorResponse

	// Send coordinator to Participant A
	client, err := rpc.Dial("tcp", addrParticipantA)
	if err != nil {
		fmt.Printf("Error sending C address to P-A: %v\n", err)
		return
	}
	if err := client.Call("Node.ParticipantConnectToCoordinator", &req, &res); err != nil {
		fmt.Printf("Error sending C address to P-A: %v\n", err)
		return
	}
	client.Close()

	// Send coordinator to Participant A
	client, err = rpc.Dial("tcp", addrParticipantB)
	if err != nil {
		fmt.Printf("Error sending C address to P-A: %v\n", err)
		return
	}
	if err := client.Call("Node.ParticipantConnectToCoordinator", &req, &res); err != nil {
		fmt.Printf("Error sending C address to P-A: %v\n", err)
		return
	}
	client.Close()

	nodesInfo := []string{
		"Coordinator: " + addrCoordinator,
		"Participant A: " + addrParticipantA,
		"Participant B: " + addrParticipantB,
	}

	// Store node data to file for client to read from
	err = utils.WriteNodeInfoToFile(nodesInfo, "nodes.txt")
	if err != nil {
		fmt.Printf("Error writing node info to file: %v\n", err)
		return
	}

	select {}

}

func startClient() {
	var client *rpc.Client
	scanner := bufio.NewScanner(os.Stdin)

	connectToServer := func() {
		servers, err := utils.ReadNodeInfoFromFile("nodes.txt")
		if err != nil {
			fmt.Printf("Error reading server info: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Available servers:")
		for i, server := range servers {
			fmt.Printf("%d: %s\n", i+1, server)
		}
		fmt.Print("Choose a server to connect to: ")
		var choice int
		fmt.Scanln(&choice)
		if choice < 1 || choice > len(servers) {
			fmt.Println("Invalid choice")
			os.Exit(1)
		}
		selectedServer := servers[choice-1]
		parts := strings.Split(selectedServer, ":")
		if len(parts) < 3 {
			fmt.Println("Invalid server format")
			os.Exit(1)
		}
		IPAddress := strings.TrimSpace(parts[len(parts)-2])
		Port := strings.TrimSpace(parts[len(parts)-1])
		fmt.Printf("Connecting to %s:%s...\n", IPAddress, Port)
		client, err = rpc.Dial("tcp", fmt.Sprintf("%s:%s", IPAddress, Port))
		if err != nil {
			fmt.Printf("Error dialing RPC server: %v\n", err)
			os.Exit(1)
		}
	}

	connectToServer() // Initial connection

	fmt.Println("Enter commands (get 'help' to see full options):")
	for {
		fmt.Print("> ")
		scanner.Scan()
		input := scanner.Text()

		if input == "exit" {

			break
		}

		parts := strings.SplitN(input, " ", 2)
		command := parts[0]

		// Process commands
		switch command {
		case "ping":
			var req node.PingRequest
			var res node.PingResponse
			if err := client.Call("Node.Ping", &req, &res); err != nil {
				fmt.Printf("Error calling RPC method: %v\n", err)
				continue
			}
			fmt.Println(res.Message)
		case "list":
			var req node.ListParticipantsRequest
			var res node.ListParticipantsResponse
			if err := client.Call("Node.ListParticipants", &req, &res); err != nil {
				fmt.Printf("Error calling RPC method: %v\n", err)
				continue
			}
			if len(res.Participants) == 0 {
				fmt.Println("No participants found.")
			} else {
				fmt.Println("List of Participants:")
				for _, participant := range res.Participants {
					fmt.Println(" -", participant)
				}
			}
		case "transfer":
			var typeReq node.GetInfoRequest
			var typeRes node.GetInfoResponse
			if err := client.Call("Node.GetInfo", &typeReq, &typeRes); err != nil {
				fmt.Printf("Error calling RPC method: %v\n", err)
				continue
			}
			if typeRes.Type != "Participant" {
				fmt.Printf("Transfer error: must be participant to initiate transfer\n")
				continue
			}
			var listReq node.ListParticipantsRequest
			var listRes node.ListParticipantsResponse
			if err := client.Call("Node.ListParticipants", &listReq, &listRes); err != nil {
				fmt.Printf("Error calling RPC method: %v\n", err)
				continue
			}
			if len(listRes.Participants) == 0 {
				fmt.Println("No participants available for transfer.")
				continue
			}
			fmt.Println("Select a participant to transfer to:")
			for i, participant := range listRes.Participants {
				selfLabel := ""
				if participant == typeRes.Name {
					selfLabel = " (self)"
				}
				fmt.Printf("%d: %s%s\n", i+1, participant, selfLabel)
			}
			var participantChoice int
			fmt.Print("Enter participant number: ")
			fmt.Scanln(&participantChoice)
			if participantChoice < 1 || participantChoice > len(listRes.Participants) {
				fmt.Println("Invalid participant choice")
				continue
			}
			targetAddr := listRes.Participants[participantChoice-1]
			// Check if the selected participant is the client itself
			if targetAddr == typeRes.Name {
				fmt.Println("Cannot transfer to self. Please select a different participant.")
				continue
			}
			var amount float64
			fmt.Print("Enter the amount to transfer: ")
			fmt.Scanln(&amount)
			if amount <= 0 {
				fmt.Println("Invalid amount")
				continue
			}

			var req node.ParticipantInitiateTransferRequest = node.ParticipantInitiateTransferRequest{
				TargetAddr: targetAddr,
				Amount:     amount,
			}
			var res node.ParticipantInitiateTransferResponse
			if err := client.Call("Node.StartTransfer", &req, &res); err != nil {
				fmt.Printf("Error calling RPC method: %v\n", err)
				continue
			}
			fmt.Println("Transfer request initiated")
		case "switch":
			if client != nil {
				client.Close()
			}
			connectToServer() // Connect to a new server
		case "bal":
			var req node.CheckFundsRequest
			var res node.CheckFundsResponse
			if err := client.Call("Node.CheckFunds", &req, &res); err != nil {
				fmt.Printf("Error calling RPC method: %v\n", err)
				continue
			}
			fmt.Printf("Balance: %.2f\n", res.Balance)
		case "deposit":
			if len(parts) != 2 {
				fmt.Printf("Usage: deposit <amout>")
				continue
			}
			amt, err := strconv.ParseFloat(parts[1], 64)
			if err != nil {
				fmt.Printf("Error parsing amount: %v", err)
				continue
			}
			var req node.DepositRequest = node.DepositRequest{
				Amount: amt,
			}
			var res node.DepositResponse
			if err := client.Call("Node.Deposit", &req, &res); err != nil {
				fmt.Printf("Error calling RPC method: %v\n", err)
				continue
			}
			fmt.Printf("Deposited %.2f\n", amt)
		default:
			fmt.Println("Unknown command:", input)
		}
	}
}

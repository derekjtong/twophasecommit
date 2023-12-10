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
	var currentAddr string
	var currentName string
	var currentType string
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
		client, err = rpc.Dial("tcp", fmt.Sprintf("%s:%s", IPAddress, Port))
		if err != nil {
			fmt.Printf("Error dialing RPC server: %v\n", err)
			os.Exit(1)
		}
		var req node.GetInfoRequest
		var res node.GetInfoResponse
		if err := client.Call("Node.GetInfo", &req, &res); err != nil {
			fmt.Printf("Error callign RPC method: %v\n", err)
			os.Exit(1)
		}
		currentAddr = res.Addr
		currentName = res.Name
		currentType = res.Type
		if currentType == "Participant" {
			fmt.Printf("Connected to %s-%s\n", currentType, currentName)
		} else {
			fmt.Printf("Connected to %s\n", currentType)
		}
	}

	getBalance := func() float64 {
		var req node.GetBalanceRequest
		var res node.GetBalanceResponse
		if err := client.Call("Node.GetBalance", &req, &res); err != nil {
			fmt.Printf("Error calling RPC method: %v\n", err)
		}
		return res.Balance
	}

	connectToServer() // Initial connection setup

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

		switch command {
		case "ping":
			// Ping current server
			var req node.PingRequest
			var res node.PingResponse
			if err := client.Call("Node.Ping", &req, &res); err != nil {
				fmt.Printf("Error calling RPC method: %v\n", err)
				continue
			}
			fmt.Println(res.Message)
		case "list":
			// List participants
			var req node.ListParticipantsRequest
			var res node.ListParticipantsResponse
			if err := client.Call("Node.ListParticipants", &req, &res); err != nil {
				fmt.Printf("Error calling RPC method: %v\n", err)
				continue
			}
			if len(res.Names) == 0 {
				fmt.Println("No participants found.")
			} else {
				fmt.Println("List of Participants:")
				for i, name := range res.Names {
					address := res.Addresses[i] // Get the corresponding address
					selfLabel := ""
					if address == currentAddr {
						selfLabel = " (self)"
					}
					fmt.Printf("%d: %s - %s%s\n", i+1, name, address, selfLabel)
				}
			}
		case "send":
			var listReq node.ListParticipantsRequest
			var listRes node.ListParticipantsResponse
			if err := client.Call("Node.ListParticipants", &listReq, &listRes); err != nil {
				fmt.Printf("Error calling RPC method: %v\n", err)
				continue
			}
			if len(listRes.Names) == 0 {
				fmt.Println("No participants available to send to.")
				continue
			}
			fmt.Println("Select a participant to send to:")
			for i, name := range listRes.Names {
				address := listRes.Addresses[i] // Get the corresponding address
				selfLabel := ""
				if address == currentAddr {
					selfLabel = " (self)"
				}
				fmt.Printf("%d: %s - %s%s\n", i+1, name, address, selfLabel)
			}
			var participantChoice int
			fmt.Print("Enter participant number: ")
			fmt.Scanln(&participantChoice)
			if participantChoice < 1 || participantChoice > len(listRes.Names) {
				fmt.Println("Invalid participant choice")
				continue
			}
			targetAddr := listRes.Addresses[participantChoice-1]
			targetName := listRes.Names[participantChoice-1]
			// Check if the selected participant is the client itself
			if targetAddr == currentAddr {
				fmt.Println("Cannot send to self. Please select a different participant.")
				continue
			}
			var amount float64
			fmt.Print("Enter the amount to send: ")
			fmt.Scanln(&amount)
			if amount <= 0 {
				fmt.Println("Invalid amount")
				continue
			}
			var req node.ClientParticipantTransactionRequest = node.ClientParticipantTransactionRequest{
				TargetAddr:      targetAddr,
				TargetName:      targetName,
				TargetOperation: "add",
				TargetAmount:    amount,
				SelfOperation:   "subtract",
				SelfAmount:      amount,
			}
			var res node.ClientParticipantTransactionResponse
			if err := client.Call("Node.ClientParticipantTransaction", &req, &res); err != nil {
				fmt.Printf("Error calling RPC method: %v\n", err)
				continue
			}
			fmt.Printf("Sent %.2f from %s to %s\n", amount, currentName, targetName)
		case "switch":
			// Connect to a new server
			if client != nil {
				client.Close()
			}
			connectToServer()
		case "bal":
			fmt.Printf("Balance: %.2f\n", getBalance())
		case "deposit":
			// Deposit money
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
		case "transaction":
			// Get the list of participants
			var listReq node.ListParticipantsRequest
			var listRes node.ListParticipantsResponse
			if err := client.Call("Node.ListParticipants", &listReq, &listRes); err != nil {
				fmt.Printf("Error calling RPC method: %v\n", err)
				continue
			}

			if len(listRes.Names) < 2 {
				fmt.Println("At least two participants are required for a transaction.")
				continue
			}

			// Function to parse operation input, can be +50 or -50
			parseOperation := func(input string) (string, float64, error) {
				var operation string
				var amount float64
				var err error

				if strings.HasPrefix(input, "+") {
					operation = "add"
					amount, err = strconv.ParseFloat(input[1:], 64)
				} else if strings.HasPrefix(input, "-") {
					operation = "subtract"
					amount, err = strconv.ParseFloat(input[1:], 64)
				} else if strings.HasPrefix(input, "*") {
					operation = "multiply"
					amount, err = strconv.ParseFloat(input[1:], 64)
				} else {
					err = fmt.Errorf("invalid operation format")
				}

				return operation, amount, err
			}

			fmt.Printf("Enter operatin and amount for the currently connected participant (e.g., +50, -30, *1.2): ")
			var senderOpInput string
			fmt.Scanln(&senderOpInput)
			var senderOperation, senderAmount, err = parseOperation(senderOpInput)
			if err != nil {
				fmt.Printf("Error parsing operation: %v\n", err)
				continue
			}

			fmt.Println("Select a participant to send to:")
			for i, name := range listRes.Names {
				address := listRes.Addresses[i] // Get the corresponding address
				selfLabel := ""
				if address == currentAddr {
					selfLabel = " (self)"
				}
				fmt.Printf("%d: %s - %s%s\n", i+1, name, address, selfLabel)
			}
			fmt.Print("Enter participant number: ")
			var participantChoice int
			fmt.Scanln(&participantChoice)
			if participantChoice < 1 || participantChoice > len(listRes.Names) {
				fmt.Println("Invalid participant choice")
				continue
			}
			targetAddr := listRes.Addresses[participantChoice-1]
			targetName := listRes.Names[participantChoice-1]
			if targetAddr == currentAddr {
				fmt.Println("Cannot send to self. Please select a different target participant.")
				continue
			}

			fmt.Printf("Enter operation and amount for target participant (e.g., +50, -30, *1.2): ")
			var targetOpInput string
			fmt.Scanln(&targetOpInput)
			targetOperation, targetAmount, err := parseOperation(targetOpInput)
			if err != nil {
				fmt.Printf("Error parsing operation: %v\n", err)
				continue
			}

			var req node.ClientParticipantTransactionRequest = node.ClientParticipantTransactionRequest{
				TargetAddr:      targetAddr,
				TargetName:      targetName,
				TargetOperation: targetOperation,
				TargetAmount:    targetAmount,
				SelfOperation:   senderOperation,
				SelfAmount:      senderAmount,
			}
			var res node.ClientParticipantTransactionResponse
			if err := client.Call("Node.ClientParticipantTransaction", &req, &res); err != nil {
				fmt.Printf("Error calling RPC method: %v\n", err)
				continue
			}
			fmt.Printf("Connected participant balance %.2f, target participant balance %.2f.\n", senderAmount, targetAmount)

		default:
			fmt.Println("Unknown command:", input)
		}
	}
}

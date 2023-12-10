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

	// Select server by number
	selectedServer := servers[choice-1]
	parts := strings.Split(selectedServer, ":")
	if len(parts) < 3 {
		fmt.Println("Invalid server format")
		os.Exit(1)
	}

	// Second to last is IP address, last is port
	IPAddress := strings.TrimSpace(parts[len(parts)-2])
	Port := strings.TrimSpace(parts[len(parts)-1])

	fmt.Printf("Connecting to %s:%s...\n", IPAddress, Port)
	client, err := rpc.Dial("tcp", fmt.Sprintf("%s:%s", IPAddress, Port))
	if err != nil {
		fmt.Printf("Error dialing RPC server: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Test the connection with a ping
	var request node.PingRequest
	var response node.PingResponse
	if err := client.Call("Node.Ping", &request, &response); err != nil {
		fmt.Printf("Error calling RPC method: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Connected to %v!\n", response.Name)

	runCLI(client)
}

func runCLI(client *rpc.Client) {
	scanner := bufio.NewScanner(os.Stdin)
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
			if len(parts) < 3 {
				fmt.Println("Usage: transfer <TargetAddr> <Amount>")
				continue
			}
			targetAddr := parts[1]
			amount, err := strconv.Atoi(parts[2])
			if err != nil {
				fmt.Printf("Invalid amount: %v\n", err)
				continue
			}

			var req node.StartTransferRequest = node.StartTransferRequest{
				TargetAddr: targetAddr,
				Amount:     amount,
			}
			var res node.StartTransferResponse
			if err := client.Call("Node.StartTransfer", &req, &res); err != nil {
				fmt.Printf("Error calling RPC method: %v\n", err)
				continue
			}
			fmt.Println("Transfer request initiated.")
		default:
			fmt.Println("Unknown command:", input)
		}
	}
}

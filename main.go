package main

import (
	"bufio"
	"fmt"
	"net/rpc"
	"os"
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
	var req = node.SetCoordinatorRequest{Addr: addrCoordinator}
	var res node.SetCoordinatorResponse

	// Send coordinator to Participant A
	client, err := rpc.Dial("tcp", addrParticipantA)
	if err != nil {
		fmt.Printf("Error sending C address to P-A: %v\n", err)
		return
	}
	if err := client.Call("Node.SetCoordinator", &req, &res); err != nil {
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
	if err := client.Call("Node.SetCoordinator", &req, &res); err != nil {
		fmt.Printf("Error sending C address to P-A: %v\n", err)
		return
	}
	client.Close()

	select {}

}

func startClient() {
	fmt.Print("Starting Client!\nNode IP address: (defaulting to 127.0.0.1)\n")
	var IPAddress string = "127.0.0.1"
	// fmt.Scanln(&IPAddress)
	fmt.Print("Node port number: ")
	var Port int
	fmt.Scanln(&Port)
	fmt.Printf("Connecting to %s:%d...\n", IPAddress, Port)
	client, err := rpc.Dial("tcp", fmt.Sprintf("%s:%d", IPAddress, Port))
	if err != nil {
		fmt.Printf("Error dialing RPC server: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	var request node.PingRequest
	var response node.PingResponse
	if err := client.Call("Node.Ping", &request, &response); err != nil {
		fmt.Printf("Error calling RPC method: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Connected to node %v!\n", response.Name)

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
		// var argument string
		// if len(parts) > 1 {
		// 	argument = parts[1]
		// }

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
		default:
			fmt.Println("Unknown command:", input)
		}
	}
}

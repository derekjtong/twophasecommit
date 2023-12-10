package utils

import (
	"fmt"
	"net"
	"net/rpc"
	"time"
	"twophasecommit/node"
)

// Find available port on system
func FindAvailablePort() (int, error) {
	// Find a free port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	// Get the allocated port
	address := listener.Addr().String()
	_, portString, err := net.SplitHostPort(address)
	if err != nil {
		return 0, err
	}

	port, err := net.LookupPort("tcp", portString)
	if err != nil {
		return 0, err
	}

	return port, nil
}

func WaitForServerReady(address string) error {
	// Blocked until server ready or timeout
	var backoff time.Duration = 100
	const maxBackoff = 5 * time.Second
	const maxRetries = 10
	var timeout time.Duration = 5 * time.Second
	startTime := time.Now()

	for retries := 0; retries < maxRetries; retries++ {
		client, err := rpc.Dial("tcp", address)
		if err == nil {
			var req node.HealthCheckRequest
			var res node.HealthCheckResponse
			err = client.Call("Node.HealthCheck", &req, &res)
			client.Close()
			if err == nil && res.Status == "OK" {
				return nil
			}
		}

		if time.Since(startTime) > timeout {
			return fmt.Errorf("server at %s did not become ready within %v", address, timeout)
		}

		if backoff < maxBackoff {
			backoff *= 2
		}
		time.Sleep(backoff)
	}
	return fmt.Errorf("server at %s did not become ready after %d attemps", address, maxRetries)
}

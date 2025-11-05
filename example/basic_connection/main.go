package main

import (
	"fmt"
	"log"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/mtls"
)

// This example demonstrates how to establish a basic connection to the Malice Network server
func main() {
	// Load configuration from auth file
	config, err := mtls.ReadConfig("../../../server/admin_127.0.0.1.auth")
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	// Establish connection
	conn, err := mtls.Connect(config)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Initialize server state
	server, err := client.NewServerStatus(conn, config)
	if err != nil {
		log.Fatalf("Failed to initialize server state: %v", err)
	}

	// Get basic server information
	fmt.Printf("Connected to server: %s\n", config.Address())
	fmt.Printf("Server version: %s\n", server.Info.Version)
	fmt.Printf("Server OS/Arch: %s/%s\n", server.Info.Os, server.Info.Arch)

	// List all clients
	fmt.Printf("\nConnected clients: %d\n", len(server.Clients))
	for _, c := range server.Clients {
		fmt.Printf("  - %s (Type: %s, Online: %v)\n", c.Name, c.Type, c.Online)
	}

	// List all listeners
	fmt.Printf("\nActive listeners: %d\n", len(server.Listeners))
	for _, listener := range server.Listeners {
		fmt.Printf("  - %s (IP: %s, Active: %v)\n", listener.Id, listener.Ip, listener.Active)
	}

	// List all sessions
	fmt.Printf("\nActive sessions: %d\n", len(server.Sessions))
	for _, session := range server.Sessions {
		target := session.Target
		if target == "" && session.Os != nil {
			target = fmt.Sprintf("%s/%s", session.Os.Name, session.Os.Arch)
		}
		fmt.Printf("  - %s (%s, Alive: %v)\n", session.SessionId, target, session.IsAlive)
	}
}

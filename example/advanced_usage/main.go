package main

import (
	"fmt"
	"log"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/mtls"
)

func main() {
	config, err := mtls.ReadConfig("../../../server/admin_127.0.0.1.auth")
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	conn, err := mtls.Connect(config)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	server, err := client.NewServerStatus(conn, config)
	if err != nil {
		log.Fatalf("Failed to initialize server state: %v", err)
	}

	aliveSessions := server.AlivedSessions()
	if len(aliveSessions) == 0 {
		log.Fatal("No alive sessions found")
	}

	session := server.AddSession(aliveSessions[0])
	fmt.Printf("Using session: %s\n", session.SessionId)

	// Example 1: Session context with custom values
	fmt.Println("\n=== Session Context ===")
	sessionWithValue, err := session.WithValue("custom_key", "custom_value")
	if err != nil {
		log.Fatalf("Failed to create session with value: %v", err)
	}

	value, err := sessionWithValue.Value("custom_key")
	if err != nil {
		log.Printf("Failed to get value: %v", err)
	} else {
		fmt.Printf("Retrieved value: %s\n", value)
	}

	// Example 2: Check session capabilities
	fmt.Println("\n=== Session Capabilities ===")
	fmt.Printf("Available modules: %d\n", len(session.Modules))

	if session.HasDepend("execute") {
		fmt.Println("✓ Session has execute module")
	}

	// Example 3: Clone session
	fmt.Println("\n=== Clone Session ===")
	sdkSession := session.Clone(consts.CalleeSDK)
	fmt.Printf("Original callee: %s\n", session.Callee)
	fmt.Printf("Cloned callee: %s\n", sdkSession.Callee)

	// Example 4: Observer pattern
	fmt.Println("\n=== Observer Pattern ===")
	observerId := server.AddObserver(session)
	fmt.Printf("Observer added: %s\n", observerId)
	defer server.RemoveObserver(observerId)

	// Example 5: Active target management
	fmt.Println("\n=== Active Target ===")
	server.ActiveTarget.Set(session)
	activeSession := server.ActiveTarget.Get()
	if activeSession != nil {
		fmt.Printf("Active session: %s\n", activeSession.SessionId)
	}

	server.ActiveTarget.Background()
	fmt.Println("Session backgrounded")

	fmt.Println("\nAdvanced usage examples completed!")
}

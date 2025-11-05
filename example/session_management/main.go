package main

import (
	"context"
	"fmt"
	"log"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/mtls"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
)

// This example demonstrates how to manage sessions
func main() {
	// Setup connection (see 01_basic_connection.go for details)
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

	// Get all sessions
	sessions, err := server.Rpc.GetSessions(context.Background(), &clientpb.SessionRequest{
		All: false, // false = only alive sessions, true = all sessions
	})
	if err != nil {
		log.Fatalf("Failed to get sessions: %v", err)
	}

	fmt.Printf("Found %d sessions\n", len(sessions.Sessions))

	// Iterate through sessions
	for _, sess := range sessions.Sessions {
		fmt.Printf("\nSession: %s\n", sess.SessionId)
		fmt.Printf("  Target: %s\n", sess.Target)
		if sess.Os != nil {
			fmt.Printf("  OS: %s/%s\n", sess.Os.Name, sess.Os.Arch)
		}
		if sess.Process != nil {
			fmt.Printf("  Process: %s (PID: %d)\n", sess.Process.Name, sess.Process.Pid)
		}
		fmt.Printf("  Alive: %v\n", sess.IsAlive)
		fmt.Printf("  Modules: %v\n", sess.Modules)
	}

	// Get a specific session by ID
	if len(sessions.Sessions) > 0 {
		sessionId := sessions.Sessions[0].SessionId
		session, err := server.Rpc.GetSession(context.Background(), &clientpb.SessionRequest{
			SessionId: sessionId,
		})
		if err != nil {
			log.Fatalf("Failed to get session: %v", err)
		}

		fmt.Printf("\nDetailed session info for %s:\n", session.SessionId)
		fmt.Printf("  Group: %s\n", session.GroupName)
		fmt.Printf("  Note: %s\n", session.Note)
		fmt.Printf("  Addons: %d\n", len(session.Addons))
		for _, addon := range session.Addons {
			fmt.Printf("    - %s (depends on: %s)\n", addon.Name, addon.Depend)
		}
	}

	// Update sessions (refresh from server)
	err = server.UpdateSessions(false)
	if err != nil {
		log.Fatalf("Failed to update sessions: %v", err)
	}

	// Get only alive sessions
	aliveSessions := server.AlivedSessions()
	fmt.Printf("\nAlive sessions: %d\n", len(aliveSessions))
}

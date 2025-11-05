package main

import (
	"context"
	"fmt"
	"log"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/mtls"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
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

	fmt.Println("Starting event stream...")
	eventStream, err := server.Rpc.Events(context.Background(), &clientpb.Empty{})
	if err != nil {
		log.Fatalf("Failed to start event stream: %v", err)
	}

	fmt.Println("Listening for events (press Ctrl+C to exit)...")
	for {
		event, err := eventStream.Recv()
		if err != nil {
			log.Printf("Event stream error: %v", err)
			return
		}

		fmt.Printf("[Event] Type: %s, Op: %s\n", event.Type, event.Op)
		if event.Session != nil {
			fmt.Printf("  Session: %s\n", event.Session.SessionId)
		}
	}
}

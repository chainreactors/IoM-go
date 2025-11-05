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

	aliveSessions := server.AlivedSessions()
	if len(aliveSessions) == 0 {
		log.Fatal("No alive sessions found")
	}

	session := server.AddSession(aliveSessions[0])
	fmt.Printf("Using session: %s (%s)\n", session.SessionId, session.Target)

	// Get task results
	tasks, err := server.Rpc.GetTasks(context.Background(), &clientpb.TaskRequest{
		SessionId: session.SessionId,
	})
	if err != nil {
		log.Fatalf("Failed to get tasks: %v", err)
	}

	fmt.Printf("\nTotal tasks: %d\n", len(tasks.Tasks))
	for i, task := range tasks.Tasks {
		if i >= 5 {
			fmt.Printf("... and %d more tasks\n", len(tasks.Tasks)-5)
			break
		}
		fmt.Printf("\nTask %d (%s):\n", task.TaskId, task.Type)
		fmt.Printf("  Session: %s\n", task.SessionId)
	}
}

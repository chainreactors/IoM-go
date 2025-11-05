package main

import (
	"context"
	"fmt"
	"log"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/mtls"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
)

// This example demonstrates how to manage listeners and pipelines
func main() {
	// Setup connection
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

	// List all listeners
	fmt.Println("=== Listeners ===")
	listeners, err := server.Rpc.GetListeners(context.Background(), &clientpb.Empty{})
	if err != nil {
		log.Fatalf("Failed to get listeners: %v", err)
	}

	for _, listener := range listeners.Listeners {
		fmt.Printf("\nListener: %s\n", listener.Id)
		fmt.Printf("  IP: %s\n", listener.Ip)
		fmt.Printf("  Active: %v\n", listener.Active)
		if listener.Pipelines != nil {
			fmt.Printf("  Pipelines: %d\n", len(listener.Pipelines.Pipelines))
		}
	}

	// List all pipelines
	fmt.Println("\n=== Pipelines ===")
	pipelines, err := server.Rpc.ListPipelines(context.Background(), &clientpb.Listener{})
	if err != nil {
		log.Fatalf("Failed to list pipelines: %v", err)
	}

	for _, pipeline := range pipelines.Pipelines {
		fmt.Printf("\nPipeline: %s\n", pipeline.Name)
		fmt.Printf("  Listener: %s\n", pipeline.ListenerId)
		fmt.Printf("  Enable: %v\n", pipeline.Enable)
		if pipeline.Tls != nil {
			fmt.Printf("  TLS: %v\n", pipeline.Tls.Enable)
		}
	}

	// Example: Register a new TCP pipeline
	fmt.Println("\n=== Registering new TCP pipeline ===")
	newPipeline := &clientpb.Pipeline{
		Name:       "example_tcp",
		ListenerId: "tcp",
		Body: &clientpb.Pipeline_Tcp{
			Tcp: &clientpb.TCPPipeline{
				Host: "0.0.0.0",
				Port: 5003,
			},
		},
		Enable: true,
	}

	_, err = server.Rpc.RegisterPipeline(context.Background(), newPipeline)
	if err != nil {
		log.Printf("Failed to register pipeline: %v", err)
	} else {
		fmt.Println("Pipeline registered successfully")
	}

	// Example: Start a pipeline
	fmt.Println("\n=== Starting pipeline ===")
	_, err = server.Rpc.StartPipeline(context.Background(), &clientpb.CtrlPipeline{
		Name:       "example_tcp",
		ListenerId: "tcp",
	})
	if err != nil {
		log.Printf("Failed to start pipeline: %v", err)
	} else {
		fmt.Println("Pipeline started successfully")
	}

	// Example: Stop a pipeline
	fmt.Println("\n=== Stopping pipeline ===")
	_, err = server.Rpc.StopPipeline(context.Background(), &clientpb.CtrlPipeline{
		Name:       "example_tcp",
		ListenerId: "tcp",
	})
	if err != nil {
		log.Printf("Failed to stop pipeline: %v", err)
	} else {
		fmt.Println("Pipeline stopped successfully")
	}

	fmt.Println("\nListener management example completed!")
}

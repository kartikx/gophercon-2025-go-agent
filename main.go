package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/anthropics/anthropic-sdk-go"
)

func main() {
	// Check if ANTHROPIC_API_KEY is set
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Println("ERROR: ANTHROPIC_API_KEY environment variable is not set")
		os.Exit(1)
	}

	// Get agent type from environment variable
	agentType := os.Getenv("AGENT_TYPE")
	if agentType == "" {
		agentType = "doc" // default to doc agent
	}

	// Get port from environment variable
	portStr := os.Getenv("PORT")
	if portStr == "" {
		portStr = "8080" // default port
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		fmt.Printf("Invalid PORT environment variable: %s\n", portStr)
		os.Exit(1)
	}

	client := anthropic.NewClient()

	var agent *Agent

	switch agentType {
	case "doc":
		agent = NewDocAgent(&client)
		agent.port = port // override the default port
	case "coder":
		agent = NewCoderAgent(&client)
		agent.port = port // override the default port
	default:
		fmt.Printf("Unknown AGENT_TYPE: %s. Valid values are 'doc' or 'coder'.\n", agentType)
		os.Exit(1)
	}

	fmt.Printf("Starting %s agent on port %d\n", agentType, port)

	// Start the agent's HTTP server
	agent.Start()

	// Run the agent (this will block and handle requests)
	agent.Run(context.Background())
}
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/rumpl/cagent/agent"
	"github.com/rumpl/cagent/config"
	goOpenAI "github.com/sashabaranov/go-openai"
)

func main() {
	// Parse command line flags
	configFile := flag.String("config", "agent.yaml", "Path to the configuration file")
	agentName := flag.String("agent", "root", "Name of the agent to run")
	initialPrompt := flag.String("prompt", "", "Initial prompt to send to the agent")
	help := flag.Bool("h", false, "Display help message")
	flag.Parse()

	// If help flag is present, print usage and exit
	if *help {
		printUsage()
		return
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Get working directory for relative file paths
	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}

	// Create a context that can be canceled
	ctx := context.Background()

	// Create the agent
	a, err := agent.NewAgent(cfg, *agentName, workingDir)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Create a runtime for the agent
	runtime, err := agent.NewRuntime(cfg, a, workingDir)
	if err != nil {
		log.Fatalf("Failed to create runtime: %v", err)
	}

	messages := []goOpenAI.ChatCompletionMessage{}
	// The runtime will handle the initial prompt

	// Run the agent runtime
	if err := runtime.Run(ctx, messages, *initialPrompt); err != nil {
		log.Fatalf("Agent error: %v", err)
	}
}

// printUsage prints the usage information
func printUsage() {
	execName := filepath.Base(os.Args[0])
	fmt.Printf("Usage: %s [options]\n\n", execName)
	fmt.Printf("Options:\n")
	fmt.Printf("  -config <path>   Path to the configuration file (default: agent.yaml)\n")
	fmt.Printf("  -agent <n>    Name of the agent to run (default: root)\n")
	fmt.Printf("  -prompt <text>   Initial prompt to send to the agent\n")
	fmt.Printf("  -h               Display this help message\n")
	fmt.Printf("\nExample:\n")
	fmt.Printf("  %s -config custom.yaml -agent containerize -prompt \"Containerize my Node.js app\"\n", execName)
}

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger.Info("Starting MCP test client")

	// Create SSE client using existing pattern from cagent
	mcpClient, err := client.NewSSEMCPClient("http://localhost:8080/mcp/sse")
	if err != nil {
		logger.Error("Failed to create MCP client", "error", err)
		os.Exit(1)
	}
	defer mcpClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Start the client connection
	logger.Info("Starting MCP client connection")
	if err := mcpClient.Start(ctx); err != nil {
		logger.Error("Failed to start MCP client", "error", err)
		os.Exit(1)
	}

	// Initialize with proper handshake
	logger.Info("Initializing MCP client")
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "cagent-test-client",
		Version: "1.0.0",
	}

	initResult, err := mcpClient.Initialize(ctx, initRequest)
	if err != nil {
		logger.Error("Failed to initialize MCP client", "error", err)
		os.Exit(1)
	}
	logger.Info("MCP client initialized successfully", 
		"server_name", initResult.ServerInfo.Name,
		"server_version", initResult.ServerInfo.Version)

	fmt.Println("Connected! Testing agent reference formatting...")

	// Test 1: List store agents (may be empty or have existing agents)
	fmt.Println("\n=== Test 1: List store agents (before pull) ===")
	request1 := mcp.CallToolRequest{}
	request1.Params.Name = "list_agents"
	request1.Params.Arguments = map[string]interface{}{
		"source": "store",
	}
	result1, err := mcpClient.CallTool(ctx, request1)
	if err != nil {
		logger.Error("Failed to list store agents", "error", err)
		os.Exit(1)
	}
	printResult("Store agents before pull", result1)
	fmt.Println("Look for 'agent_ref' field - for store agents, this should be full image reference with tag")

	// Test 2: List file agents
	fmt.Println("\n=== Test 2: List file agents ===")
	request2 := mcp.CallToolRequest{}
	request2.Params.Name = "list_agents"
	request2.Params.Arguments = map[string]interface{}{
		"source": "files",
	}
	result2, err := mcpClient.CallTool(ctx, request2)
	if err != nil {
		logger.Error("Failed to list file agents", "error", err)
		os.Exit(1)
	}
	printResult("File agents", result2)
	fmt.Println("Look for 'agent_ref' field - for file agents, this should be the relative path from agents directory")

	// Test 3: List all agents
	fmt.Println("\n=== Test 3: List all agents (files + store) ===")
	request3 := mcp.CallToolRequest{}
	request3.Params.Name = "list_agents"
	request3.Params.Arguments = map[string]interface{}{
		"source": "all",
	}
	result3, err := mcpClient.CallTool(ctx, request3)
	if err != nil {
		logger.Error("Failed to list all agents", "error", err)
		os.Exit(1)
	}
	printResult("All agents", result3)

	fmt.Println("\n=== VERIFICATION: Check agent_ref format ===")
	fmt.Println("Expected for file agents: agent_ref should be relative path from agents directory")
	fmt.Println("Expected for store agents: agent_ref should be full image reference with tag")
	fmt.Println("\nTest completed successfully!")
}

func printResult(title string, result *mcp.CallToolResult) {
	if result.IsError {
		fmt.Printf("%s ERROR: %v\n", title, result.Content)
		return
	}

	if len(result.Content) == 0 {
		fmt.Printf("%s: No content\n", title)
		return
	}

	// Extract text content
	for i, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			fmt.Printf("%s [%d]: %s\n", title, i, textContent.Text)
		} else {
			fmt.Printf("%s [%d]: %v\n", title, i, content)
		}
	}
}
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

	fmt.Println("Connected! Testing store agent listing...")

	// Test 1: List store agents (should be empty initially)
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

	// Test 2: Pull the jean-laurent agent
	fmt.Println("\n=== Test 2: Pull jean-laurent agent ===")
	request2 := mcp.CallToolRequest{}
	request2.Params.Name = "pull_agent"
	request2.Params.Arguments = map[string]interface{}{
		"registry_ref": "djordjelukic1639080/jean-laurent",
	}
	result2, err := mcpClient.CallTool(ctx, request2)
	if err != nil {
		logger.Error("Failed to pull agent", "error", err)
		os.Exit(1)
	}
	printResult("Pull result", result2)

	// Test 3: List store agents again (should now show the pulled agent)
	fmt.Println("\n=== Test 3: List store agents (after pull) ===")
	request3 := mcp.CallToolRequest{}
	request3.Params.Name = "list_agents"
	request3.Params.Arguments = map[string]interface{}{
		"source": "store",
	}
	result3, err := mcpClient.CallTool(ctx, request3)
	if err != nil {
		logger.Error("Failed to list store agents after pull", "error", err)
		os.Exit(1)
	}
	printResult("Store agents after pull", result3)

	// Test 4: List all agents
	fmt.Println("\n=== Test 4: List all agents (files + store) ===")
	request4 := mcp.CallToolRequest{}
	request4.Params.Name = "list_agents"
	request4.Params.Arguments = map[string]interface{}{
		"source": "all",
	}
	result4, err := mcpClient.CallTool(ctx, request4)
	if err != nil {
		logger.Error("Failed to list all agents", "error", err)
		os.Exit(1)
	}
	printResult("All agents", result4)

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
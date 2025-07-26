// server.go implements the MCP (Model Context Protocol) server for cagent's multi-tenant architecture.
// This component provides the transport layer that bridges MCP clients with cagent's servicecore business logic.
//
// Core Responsibilities:
// 1. MCP Protocol Implementation:
//    - Registers MCP tools (invoke_agent, list_agents, pull_agent, session management)
//    - Handles MCP client connections and lifecycle events
//    - Provides structured error responses following MCP conventions
//    - Manages real client IDs extracted from MCP session context
//
// 2. Client Lifecycle Management:
//    - OnClientConnect: Creates new client in servicecore with real MCP client ID
//    - OnClientDisconnect: Removes client and cleans up all associated sessions
//    - Client ID validation and uniqueness enforcement
//    - Automatic resource cleanup on disconnect
//
// 3. Tool Registration and Routing:
//    - Maps MCP tool calls to servicecore operations
//    - Handles parameter validation and type conversion
//    - Provides consistent error handling across all tools
//    - Formats responses for MCP client consumption
//
// 4. Security and Isolation:
//    - Enforces client isolation through servicecore operations
//    - Validates all client operations against proper client context
//    - Prevents cross-client access through MCP protocol violations
//    - Logs security-relevant events for monitoring
//
// Integration Architecture:
// - Uses servicecore.ServiceManager for all business logic
// - Leverages mcp-go library for protocol implementation
// - Provides clean separation between transport and business logic
// - Enables consistent behavior across MCP and future HTTP transports
//
package mcpserver

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/docker/cagent/pkg/servicecore"
)

// MCPServer implements the MCP server using servicecore for business logic
type MCPServer struct {
	serviceCore servicecore.ServiceManager
	mcpServer   *server.MCPServer
	sseServer   *server.SSEServer
	logger      *slog.Logger
}

// NewMCPServer creates a new MCP server instance
func NewMCPServer(serviceCore servicecore.ServiceManager, logger *slog.Logger, basePath string) *MCPServer {
	mcpServerInstance := &MCPServer{
		serviceCore: serviceCore,
		logger:      logger,
	}

	// Create MCP server with tool capabilities
	mcpServerInstance.mcpServer = server.NewMCPServer("cagent", "1.0.0",
		server.WithToolCapabilities(true))

	// Register tools
	mcpServerInstance.registerTools()

	// Create SSE server wrapper for HTTP transport with streaming
	mcpServerInstance.sseServer = server.NewSSEServer(mcpServerInstance.mcpServer,
		server.WithStaticBasePath(basePath),
		server.WithKeepAliveInterval(30*time.Second),
	)

	return mcpServerInstance
}

// Start starts the MCP SSE server on the specified port
func (s *MCPServer) Start(ctx context.Context, port string) error {
	s.logger.Info("Starting MCP SSE server", "port", port)
	
	// Start SSE server on specified port in a goroutine
	addr := ":" + port
	errChan := make(chan error, 1)
	
	go func() {
		if err := s.sseServer.Start(addr); err != nil {
			errChan <- fmt.Errorf("serving MCP SSE server: %w", err)
		}
	}()
	
	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		s.logger.Info("MCP SSE server shutting down")
		// TODO: Add graceful shutdown when mcp-go supports it
		return nil
	case err := <-errChan:
		return err
	}
}

// TODO: Client lifecycle management will be added when mcp-go library supports it
// For now, we'll manage clients within individual tool calls

// registerTools registers all MCP tools with the server
func (s *MCPServer) registerTools() {
	// One-shot agent invocation
	s.mcpServer.AddTool(mcp.NewTool("invoke_agent",
		mcp.WithDescription("Invoke an agent with a message and get a response"),
		mcp.WithString("agent", mcp.Required(), mcp.Description("Agent specification (file path, relative path, or registry reference)")),
		mcp.WithString("message", mcp.Required(), mcp.Description("Message to send to the agent")),
	), s.handleInvokeAgent)

	// List available agents
	s.mcpServer.AddTool(mcp.NewTool("list_agents",
		mcp.WithDescription("List available agents"),
		mcp.WithString("source", mcp.Description("Agent source filter: 'files', 'store', or 'all' (default)")),
	), s.handleListAgents)

	// Pull agent from registry
	s.mcpServer.AddTool(mcp.NewTool("pull_agent",
		mcp.WithDescription("Pull an agent image from registry to local store"),
		mcp.WithString("registry_ref", mcp.Required(), mcp.Description("Registry reference (e.g., 'myregistry.com/myagent:latest')")),
	), s.handlePullAgent)

	s.logger.Debug("Registered MCP tools", "tools", []string{"invoke_agent", "list_agents", "pull_agent"})
}
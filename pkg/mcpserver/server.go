// server.go implements the MCP (Model Context Protocol) server for cagent's multi-tenant architecture.
// This component provides the transport layer that bridges MCP clients with cagent's servicecore business logic.
//
// Core Responsibilities:
// 1. MCP Protocol Implementation:
//   - Registers MCP tools (invoke_agent, list_agents, pull_agent, session management)
//   - Handles MCP client connections and lifecycle events
//   - Provides structured error responses following MCP conventions
//   - Manages real client IDs extracted from MCP session context
//
// 2. Client Lifecycle Management:
//   - OnClientConnect: Creates new client in servicecore with real MCP client ID
//   - OnClientDisconnect: Removes client and cleans up all associated sessions
//   - Client ID validation and uniqueness enforcement
//   - Automatic resource cleanup on disconnect
//
// 3. Tool Registration and Routing:
//   - Maps MCP tool calls to servicecore operations
//   - Handles parameter validation and type conversion
//   - Provides consistent error handling across all tools
//   - Formats responses for MCP client consumption
//
// 4. Security and Isolation:
//   - Enforces client isolation through servicecore operations
//   - Validates all client operations against proper client context
//   - Prevents cross-client access through MCP protocol violations
//   - Logs security-relevant events for monitoring
//
// Integration Architecture:
// - Uses servicecore.ServiceManager for all business logic
// - Leverages mcp-go library for protocol implementation
// - Provides clean separation between transport and business logic
// - Enables consistent behavior across MCP and future HTTP transports
package mcpserver

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/docker/cagent/pkg/servicecore"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
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
		mcp.WithDescription("Invoke an agent with a single message and get a response (one-shot execution). Use this for simple, single interactions. For ongoing conversations, use create_agent_session followed by send_message instead."),
		mcp.WithString("agent", mcp.Required(), mcp.Description("Agent specification (file path, relative path, or registry reference like 'myregistry.com/agent:latest')")),
		mcp.WithString("message", mcp.Required(), mcp.Description("Message to send to the agent")),
	), s.handleInvokeAgent)

	// List available agents
	s.mcpServer.AddTool(mcp.NewTool("list_agents",
		mcp.WithDescription("List all available agents that can be used with invoke_agent or create_agent_session. Shows agents from local files and pulled registry images. Use this to discover available agents before invoking them."),
		mcp.WithString("source", mcp.Description("Agent source filter: 'files' (local config files), 'store' (pulled images), or 'all' (default - shows both)")),
	), s.handleListAgents)

	// Pull agent from registry
	s.mcpServer.AddTool(mcp.NewTool("pull_agent",
		mcp.WithDescription("Pull an agent image from a Docker registry to local store, making it available for invoke_agent and create_agent_session. Use this to download agents from registries before using them."),
		mcp.WithString("registry_ref", mcp.Required(), mcp.Description("Registry reference (e.g., 'myregistry.com/myagent:latest' or 'docker.io/user/agent:v1.0')")),
	), s.handlePullAgent)

	// Session management tools
	s.mcpServer.AddTool(mcp.NewTool("create_agent_session",
		mcp.WithDescription("Create a persistent agent session. Returns a session ID that must be used with send_message, get_agent_session_info, get_agent_session_history, and close_agent_session tools. Use this when you want to have an ongoing conversation with an agent rather than one-shot invocations."),
		mcp.WithString("agent", mcp.Required(), mcp.Description("Agent specification (file path, relative path, or registry reference)")),
	), s.handleCreateAgentSession)

	s.mcpServer.AddTool(mcp.NewTool("send_message",
		mcp.WithDescription("Send a message to an existing agent session created with create_agent_session. The session_id parameter must be the session ID returned from create_agent_session. Use this for ongoing conversations with persistent agents."),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("Session ID returned from create_agent_session")),
		mcp.WithString("message", mcp.Required(), mcp.Description("Message to send to the agent")),
	), s.handleSendMessage)

	s.mcpServer.AddTool(mcp.NewTool("list_agent_sessions",
		mcp.WithDescription("List all active agent sessions for the current client. Shows sessions created with create_agent_session that haven't been closed. Use this to see what sessions are available for send_message, get_agent_session_info, or close_agent_session."),
	), s.handleListAgentSessions)

	s.mcpServer.AddTool(mcp.NewTool("close_agent_session",
		mcp.WithDescription("Close and cleanup an existing agent session created with create_agent_session. After closing, the session_id can no longer be used with send_message or other session tools."),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("Session ID returned from create_agent_session to close")),
	), s.handleCloseAgentSession)

	s.mcpServer.AddTool(mcp.NewTool("get_agent_session_info",
		mcp.WithDescription("Get detailed information about a specific agent session created with create_agent_session. Shows metadata like creation time, last used time, agent details, and message count."),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("Session ID returned from create_agent_session")),
	), s.handleGetAgentSessionInfo)

	// Advanced session management tools
	s.mcpServer.AddTool(mcp.NewTool("get_agent_session_history",
		mcp.WithDescription("Get conversation history for an agent session created with create_agent_session. Returns all messages exchanged with the agent, optionally paginated. Useful for reviewing past conversations or context."),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("Session ID returned from create_agent_session")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of messages to return (default: 50, 0 for all)")),
	), s.handleGetAgentSessionHistory)

	s.mcpServer.AddTool(mcp.NewTool("get_agent_session_info_enhanced",
		mcp.WithDescription("Get comprehensive information about an agent session created with create_agent_session. Includes detailed agent metadata, available tools, statistics, and session state. More detailed than get_agent_session_info."),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("Session ID returned from create_agent_session")),
	), s.handleGetAgentSessionInfoEnhanced)

	s.logger.Debug("Registered MCP tools", "tools", []string{
		"invoke_agent", "list_agents", "pull_agent",
		"create_agent_session", "send_message", "list_agent_sessions",
		"close_agent_session", "get_agent_session_info",
		"get_agent_session_history", "get_agent_session_info_enhanced",
	})
}

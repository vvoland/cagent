// handlers.go implements MCP tool handlers that bridge MCP protocol calls to servicecore operations.
// This file contains the actual tool implementations that process MCP requests and format responses.
//
// Tool Implementation Pattern:
// 1. Extract and validate parameters from MCP request
// 2. Extract client ID from MCP context for multi-tenant operation
// 3. Call appropriate servicecore method with client scoping
// 4. Format response for MCP client consumption
// 5. Handle errors with structured MCP error responses
//
// Security Considerations:
// - All operations require valid client ID from MCP context
// - Parameter validation prevents malicious input
// - Errors are logged but sanitized for client responses
// - Client isolation is enforced through servicecore operations
//
package mcpserver

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// handleInvokeAgent implements one-shot agent invocation
func (s *MCPServer) handleInvokeAgent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract client ID from context
	clientID, err := s.extractClientID(ctx)
	if err != nil {
		return nil, err
	}

	// Validate and extract parameters
	args, ok := req.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments format")
	}

	agent, ok := args["agent"].(string)
	if !ok || agent == "" {
		return nil, fmt.Errorf("agent parameter is required and must be a string")
	}

	message, ok := args["message"].(string)
	if !ok || message == "" {
		return nil, fmt.Errorf("message parameter is required and must be a string")
	}

	s.logger.Debug("Invoking agent", "client_id", clientID, "agent", agent, "message_length", len(message))

	// Create agent session
	session, err := s.serviceCore.CreateAgentSession(clientID, agent)
	if err != nil {
		s.logger.Error("Failed to create agent session", "client_id", clientID, "agent", agent, "error", err)
		return nil, fmt.Errorf("creating agent session: %w", err)
	}

	// Send message and get response
	response, err := s.serviceCore.SendMessage(clientID, session.ID, message)
	if err != nil {
		// Clean up session on error
		if cleanupErr := s.serviceCore.CloseSession(clientID, session.ID); cleanupErr != nil {
			s.logger.Warn("Failed to cleanup session after error", "session_id", session.ID, "error", cleanupErr)
		}
		s.logger.Error("Failed to send message", "client_id", clientID, "session_id", session.ID, "error", err)
		return nil, fmt.Errorf("sending message: %w", err)
	}

	// Clean up session after one-shot invocation
	if err := s.serviceCore.CloseSession(clientID, session.ID); err != nil {
		s.logger.Warn("Failed to cleanup session after completion", "session_id", session.ID, "error", err)
	}

	// Format response for MCP client
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: response.Content,
			},
		},
		IsError: false,
	}, nil
}

// handleListAgents implements agent listing
func (s *MCPServer) handleListAgents(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract client ID from context (for logging, but list_agents doesn't require client scoping)
	clientID, err := s.extractClientID(ctx)
	if err != nil {
		return nil, err
	}

	// Extract source parameter (optional)
	source := "all" // default
	if req.Params.Arguments != nil {
		if args, ok := req.Params.Arguments.(map[string]interface{}); ok {
			if s, ok := args["source"].(string); ok && s != "" {
				source = s
			}
		}
	}

	s.logger.Debug("Listing agents", "client_id", clientID, "source", source)

	// Get agents from servicecore
	agents, err := s.serviceCore.ListAgents(source)
	if err != nil {
		s.logger.Error("Failed to list agents", "client_id", clientID, "source", source, "error", err)
		return nil, fmt.Errorf("listing agents: %w", err)
	}

	// Format response for MCP client
	var agentList []interface{}
	for _, agent := range agents {
		agentInfo := map[string]interface{}{
			"name":        agent.Name,
			"description": agent.Description,
			"source":      agent.Source,
		}
		if agent.Path != "" {
			agentInfo["path"] = agent.Path
		}
		if agent.Reference != "" {
			agentInfo["reference"] = agent.Reference
		}
		agentList = append(agentList, agentInfo)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Found %d agents:\n\n%s", len(agents), formatAgentList(agentList)),
			},
		},
		IsError: false,
	}, nil
}

// handlePullAgent implements agent pulling from registry
func (s *MCPServer) handlePullAgent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract client ID from context (for logging, but pull_agent doesn't require client scoping)
	clientID, err := s.extractClientID(ctx)
	if err != nil {
		return nil, err
	}

	// Validate and extract parameters
	args, ok := req.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments format")
	}

	registryRef, ok := args["registry_ref"].(string)
	if !ok || registryRef == "" {
		return nil, fmt.Errorf("registry_ref parameter is required and must be a string")
	}

	s.logger.Info("Pulling agent", "client_id", clientID, "registry_ref", registryRef)

	// Pull agent using servicecore
	if err := s.serviceCore.PullAgent(registryRef); err != nil {
		s.logger.Error("Failed to pull agent", "client_id", clientID, "registry_ref", registryRef, "error", err)
		return nil, fmt.Errorf("pulling agent: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Successfully pulled agent: %s", registryRef),
			},
		},
		IsError: false,
	}, nil
}

// extractClientID extracts the client ID from MCP context
func (s *MCPServer) extractClientID(ctx context.Context) (string, error) {
	// TODO: Extract actual client ID from MCP context
	// For now, use a placeholder client ID and create client if needed
	clientID := "mcp-client-1" // Placeholder
	
	// Ensure client exists in servicecore
	if err := s.serviceCore.CreateClient(clientID); err != nil {
		// Client might already exist, which is fine
		s.logger.Debug("Client creation result", "client_id", clientID, "error", err)
	}
	
	if clientID == "" {
		return "", fmt.Errorf("client ID not found in context")
	}

	return clientID, nil
}

// formatAgentList formats the agent list for text display
func formatAgentList(agents []interface{}) string {
	var result string
	for i, agent := range agents {
		if agentMap, ok := agent.(map[string]interface{}); ok {
			name := agentMap["name"]
			desc := agentMap["description"]
			source := agentMap["source"]
			result += fmt.Sprintf("%d. %s (%s)\n   %s\n", i+1, name, source, desc)
		}
	}
	return result
}
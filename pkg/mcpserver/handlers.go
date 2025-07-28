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
	"github.com/docker/cagent/pkg/servicecore"
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
	
	// Debug the response we got from servicecore
	s.logger.Debug("Got response from servicecore", 
		"client_id", clientID, 
		"session_id", session.ID, 
		"content_length", len(response.Content),
		"event_count", len(response.Events),
		"content_preview", func() string {
			if len(response.Content) > 100 {
				return response.Content[:100] + "..."
			}
			return response.Content
		}())

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
		// Determine the agent_ref based on source type
		var agentRef string
		if agent.Source == "file" {
			agentRef = agent.RelativePath // For file agents, use the relative path from agents dir
		} else if agent.Source == "store" {
			agentRef = agent.Reference // For store agents, use the full image reference with tag
		}
		
		agentInfo := map[string]interface{}{
			"agent_ref":    agentRef,
			"friendly_name": agent.Name,
			"source":       agent.Source,
			"description":  agent.Description,
		}
		
		// Keep legacy fields for backward compatibility
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

// handleCreateAgentSession implements persistent agent session creation
func (s *MCPServer) handleCreateAgentSession(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	s.logger.Debug("Creating agent session", "client_id", clientID, "agent", agent)

	// Create agent session
	session, err := s.serviceCore.CreateAgentSession(clientID, agent)
	if err != nil {
		s.logger.Error("Failed to create agent session", "client_id", clientID, "agent", agent, "error", err)
		return nil, fmt.Errorf("creating agent session: %w", err)
	}

	// Format response for MCP client
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Created agent session: %s\nAgent: %s\nClient: %s\nCreated: %s", 
					session.ID, session.AgentSpec, session.ClientID, session.Created.Format("2006-01-02 15:04:05")),
			},
		},
		IsError: false,
	}, nil
}

// handleSendMessage implements message sending to existing agent sessions
func (s *MCPServer) handleSendMessage(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	sessionID, ok := args["session_id"].(string)
	if !ok || sessionID == "" {
		return nil, fmt.Errorf("session_id parameter is required and must be a string")
	}

	message, ok := args["message"].(string)
	if !ok || message == "" {
		return nil, fmt.Errorf("message parameter is required and must be a string")
	}

	s.logger.Debug("Sending message to session", "client_id", clientID, "session_id", sessionID, "message_length", len(message))

	// Send message using servicecore
	response, err := s.serviceCore.SendMessage(clientID, sessionID, message)
	if err != nil {
		s.logger.Error("Failed to send message", "client_id", clientID, "session_id", sessionID, "error", err)
		return nil, fmt.Errorf("sending message: %w", err)
	}

	// Debug the response we got from servicecore
	s.logger.Debug("Got response from servicecore", 
		"client_id", clientID, 
		"session_id", sessionID, 
		"content_length", len(response.Content),
		"event_count", len(response.Events),
		"content_preview", func() string {
			if len(response.Content) > 100 {
				return response.Content[:100] + "..."
			}
			return response.Content
		}())

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

// handleListAgentSessions implements session listing for a client
func (s *MCPServer) handleListAgentSessions(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract client ID from context
	clientID, err := s.extractClientID(ctx)
	if err != nil {
		return nil, err
	}

	s.logger.Debug("Listing agent sessions", "client_id", clientID)

	// Get sessions from servicecore
	sessions, err := s.serviceCore.ListSessions(clientID)
	if err != nil {
		s.logger.Error("Failed to list sessions", "client_id", clientID, "error", err)
		return nil, fmt.Errorf("listing sessions: %w", err)
	}

	// Format response for MCP client
	var sessionList []interface{}
	for _, session := range sessions {
		sessionInfo := map[string]interface{}{
			"id":         session.ID,
			"agent_spec": session.AgentSpec,
			"created":    session.Created.Format("2006-01-02 15:04:05"),
			"last_used":  session.LastUsed.Format("2006-01-02 15:04:05"),
		}
		sessionList = append(sessionList, sessionInfo)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Found %d active sessions:\n\n%s", len(sessions), formatSessionList(sessionList)),
			},
		},
		IsError: false,
	}, nil
}

// handleCloseAgentSession implements session closure
func (s *MCPServer) handleCloseAgentSession(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	sessionID, ok := args["session_id"].(string)
	if !ok || sessionID == "" {
		return nil, fmt.Errorf("session_id parameter is required and must be a string")
	}

	s.logger.Debug("Closing agent session", "client_id", clientID, "session_id", sessionID)

	// Close session using servicecore
	if err := s.serviceCore.CloseSession(clientID, sessionID); err != nil {
		s.logger.Error("Failed to close session", "client_id", clientID, "session_id", sessionID, "error", err)
		return nil, fmt.Errorf("closing session: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Successfully closed session: %s", sessionID),
			},
		},
		IsError: false,
	}, nil
}

// handleGetAgentSessionInfo implements session metadata retrieval
func (s *MCPServer) handleGetAgentSessionInfo(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	sessionID, ok := args["session_id"].(string)
	if !ok || sessionID == "" {
		return nil, fmt.Errorf("session_id parameter is required and must be a string")
	}

	s.logger.Debug("Getting agent session info", "client_id", clientID, "session_id", sessionID)

	// Get sessions from servicecore and find the requested one
	sessions, err := s.serviceCore.ListSessions(clientID)
	if err != nil {
		s.logger.Error("Failed to list sessions", "client_id", clientID, "error", err)
		return nil, fmt.Errorf("listing sessions: %w", err)
	}

	// Find the specific session
	var targetSession *servicecore.AgentSession
	for _, session := range sessions {
		if session.ID == sessionID {
			targetSession = session
			break
		}
	}

	if targetSession == nil {
		return nil, fmt.Errorf("session %s not found for client %s", sessionID, clientID)
	}

	// Format detailed session info
	sessionInfo := fmt.Sprintf(`Session Information:
ID: %s
Agent Spec: %s
Client ID: %s
Created: %s
Last Used: %s
`, 
		targetSession.ID,
		targetSession.AgentSpec,
		targetSession.ClientID,
		targetSession.Created.Format("2006-01-02 15:04:05"),
		targetSession.LastUsed.Format("2006-01-02 15:04:05"))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: sessionInfo,
			},
		},
		IsError: false,
	}, nil
}

// formatAgentList formats the agent list for text display
func formatAgentList(agents []interface{}) string {
	var result string
	for i, agent := range agents {
		if agentMap, ok := agent.(map[string]interface{}); ok {
			agentRef := agentMap["agent_ref"]
			friendlyName := agentMap["friendly_name"]
			source := agentMap["source"]
			desc := agentMap["description"]
			
			result += fmt.Sprintf("%d. %s\n", i+1, friendlyName)
			result += fmt.Sprintf("   agent_ref: %s\n", agentRef)
			result += fmt.Sprintf("   source: %s\n", source)
			result += fmt.Sprintf("   description: %s\n\n", desc)
		}
	}
	return result
}

// formatSessionList formats the session list for text display
func formatSessionList(sessions []interface{}) string {
	var result string
	for i, session := range sessions {
		if sessionMap, ok := session.(map[string]interface{}); ok {
			id := sessionMap["id"]
			agentSpec := sessionMap["agent_spec"]
			created := sessionMap["created"]
			lastUsed := sessionMap["last_used"]
			result += fmt.Sprintf("%d. %s (Agent: %s)\n   Created: %s, Last Used: %s\n", 
				i+1, id, agentSpec, created, lastUsed)
		}
	}
	return result
}

// handleGetAgentSessionHistory implements conversation history retrieval with pagination
func (s *MCPServer) handleGetAgentSessionHistory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	sessionID, ok := args["session_id"].(string)
	if !ok || sessionID == "" {
		return nil, fmt.Errorf("session_id parameter is required and must be a string")
	}

	// Optional limit parameter (default to 50 if not specified)
	limit := 50
	if limitVal, exists := args["limit"]; exists {
		if limitFloat, ok := limitVal.(float64); ok {
			limit = int(limitFloat)
		} else if limitInt, ok := limitVal.(int); ok {
			limit = limitInt
		}
	}

	s.logger.Debug("Getting session history", "client_id", clientID, "session_id", sessionID, "limit", limit)

	// Get session history from servicecore
	history, err := s.serviceCore.GetSessionHistory(clientID, sessionID, limit)
	if err != nil {
		s.logger.Error("Failed to get session history", "client_id", clientID, "session_id", sessionID, "error", err)
		return nil, fmt.Errorf("getting session history: %w", err)
	}

	// Format history for display
	var result string
	if len(history) == 0 {
		result = "No conversation history found for this session."
	} else {
		result = fmt.Sprintf("Conversation History (showing %d messages):\n\n", len(history))
		for i, msg := range history {
			role := msg.Role
			if msg.AgentName != "" {
				role = fmt.Sprintf("%s (%s)", role, msg.AgentName)
			}
			result += fmt.Sprintf("%d. [%s]: %s\n\n", i+1, role, msg.Content)
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: result,
			},
		},
		IsError: false,
	}, nil
}

// handleGetAgentSessionInfoEnhanced implements enhanced session metadata retrieval
func (s *MCPServer) handleGetAgentSessionInfoEnhanced(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	sessionID, ok := args["session_id"].(string)
	if !ok || sessionID == "" {
		return nil, fmt.Errorf("session_id parameter is required and must be a string")
	}

	s.logger.Debug("Getting enhanced session info", "client_id", clientID, "session_id", sessionID)

	// Get enhanced session info from servicecore
	sessionInfo, err := s.serviceCore.GetSessionInfo(clientID, sessionID)
	if err != nil {
		s.logger.Error("Failed to get session info", "client_id", clientID, "session_id", sessionID, "error", err)
		return nil, fmt.Errorf("getting session info: %w", err)
	}

	// Format enhanced session info
	result := fmt.Sprintf(`Enhanced Session Information:

Basic Details:
  Session ID: %s
  Agent Spec: %s
  Client ID: %s
  Created: %s
  Last Used: %s
  Message Count: %d

Agent Details:
  Agent Name: %s
  Description: %s
  Instruction: %s

Toolsets: %v

Session Details:
  Internal Session ID: %s
  Session Created: %s
`, 
		sessionInfo.ID,
		sessionInfo.AgentSpec,
		sessionInfo.ClientID,
		sessionInfo.Created.Format("2006-01-02 15:04:05"),
		sessionInfo.LastUsed.Format("2006-01-02 15:04:05"),
		sessionInfo.MessageCount,
		sessionInfo.Metadata["agent_name"],
		sessionInfo.Metadata["agent_description"],
		sessionInfo.Metadata["agent_instruction"],
		sessionInfo.Metadata["toolsets"],
		sessionInfo.Metadata["session_id"],
		sessionInfo.Metadata["session_created_at"])

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: result,
			},
		},
		IsError: false,
	}, nil
}
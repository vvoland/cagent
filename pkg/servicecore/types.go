// Package servicecore provides the core business logic layer for cagent's multi-tenant agent services.
// This package abstracts the complexity of agent resolution, runtime management, and session handling
// to provide a clean interface for both MCP and HTTP transport layers.
//
// Key Design Principles:
// - Multi-tenant architecture: All operations are client-scoped to ensure isolation
// - Transport-agnostic: Core business logic is independent of MCP or HTTP specifics
// - Security-first: Agent resolution is restricted to configured root directories
// - Resource management: Proper cleanup and lifecycle management for runtimes and sessions
//
// The types in this file define the core interfaces and data structures that enable:
// 1. Client isolation through explicit client IDs in all operations
// 2. Agent metadata representation across different sources (files, Docker images)
// 3. Structured response formatting with events and metadata
// 4. Session lifecycle management with proper resource tracking
package servicecore

import (
	"context"
	"time"

	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/session"
)

// DEFAULT_CLIENT_ID is used for HTTP API compatibility and existing sessions
const DEFAULT_CLIENT_ID = "__global"

// ServiceManager defines the core interface for multi-tenant agent service operations
type ServiceManager interface {
	// Client lifecycle
	CreateClient(clientID string) error
	RemoveClient(clientID string) error

	// Agent operations
	ResolveAgent(agentSpec string) (string, error)
	ListAgents(source string) ([]AgentInfo, error)
	PullAgent(registryRef string) error

	// Session operations (client-scoped)
	CreateAgentSession(clientID, agentSpec string) (*AgentSession, error)
	SendMessage(clientID, sessionID, message string) (*Response, error)
	ListSessions(clientID string) ([]*AgentSession, error)
	CloseSession(clientID, sessionID string) error

	// Advanced session operations
	GetSessionHistory(clientID, sessionID string, limit int) ([]SessionMessage, error)
	GetSessionInfo(clientID, sessionID string) (*SessionInfo, error)
}

// AgentInfo represents metadata about an available agent
type AgentInfo struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Source       string `json:"source"`                  // "file", "store"
	Path         string `json:"path,omitempty"`          // Absolute path (for internal use)
	RelativePath string `json:"relative_path,omitempty"` // Relative path from agents dir (for user reference)
	Reference    string `json:"reference,omitempty"`     // Full image reference (for store agents)
}

// Response represents a structured response from agent execution
type Response struct {
	Content  string                 `json:"content"`
	Events   []runtime.Event        `json:"events"`
	Metadata map[string]interface{} `json:"metadata"`
}

// Client represents an MCP or HTTP client session
type Client struct {
	ID            string
	AgentSessions map[string]*AgentSession
	Created       time.Time
	LastUsed      time.Time
}

// AgentSession represents a conversational session with a specific agent
type AgentSession struct {
	ID        string
	ClientID  string
	AgentSpec string
	Runtime   *runtime.Runtime
	Session   *session.Session
	Created   time.Time
	LastUsed  time.Time
}

// SessionMessage represents a message in the conversation history
type SessionMessage struct {
	AgentName string    `json:"agent_name"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// SessionInfo represents detailed information about a session
type SessionInfo struct {
	ID           string                 `json:"id"`
	ClientID     string                 `json:"client_id"`
	AgentSpec    string                 `json:"agent_spec"`
	Created      time.Time              `json:"created"`
	LastUsed     time.Time              `json:"last_used"`
	MessageCount int                    `json:"message_count"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// Store defines the interface for multi-tenant session storage
type Store interface {
	// Client operations
	CreateClient(ctx context.Context, clientID string) error
	DeleteClient(ctx context.Context, clientID string) error

	// Session operations (client-scoped)
	CreateSession(ctx context.Context, clientID string, session *AgentSession) error
	GetSession(ctx context.Context, clientID, sessionID string) (*AgentSession, error)
	ListSessions(ctx context.Context, clientID string) ([]*AgentSession, error)
	UpdateSession(ctx context.Context, clientID string, session *AgentSession) error
	DeleteSession(ctx context.Context, clientID, sessionID string) error
}

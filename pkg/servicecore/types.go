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
}

// AgentInfo represents metadata about an available agent
type AgentInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"` // "file", "store"
	Path        string `json:"path,omitempty"`
	Reference   string `json:"reference,omitempty"`
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
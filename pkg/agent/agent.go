package agent

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sync/atomic"

	"github.com/docker/cagent/pkg/memorymanager"
	"github.com/docker/cagent/pkg/model/provider"
	"github.com/docker/cagent/pkg/tools"
	"github.com/mark3labs/mcp-go/client"
)

// OAuthAuthorizationRequiredError wraps an OAuth authorization error with server information
type OAuthAuthorizationRequiredError struct {
	Err        error
	ServerURL  string
	ServerType string
}

func (e *OAuthAuthorizationRequiredError) Error() string {
	return fmt.Sprintf("OAuth authorization required for %s server '%s': %v", e.ServerType, e.ServerURL, e.Err)
}

func (e *OAuthAuthorizationRequiredError) Unwrap() error {
	return e.Err
}

// Agent represents an AI agent
type Agent struct {
	name               string
	description        string
	instruction        string
	toolsets           []tools.ToolSet
	toolsetsStarted    atomic.Bool
	models             []provider.Provider
	subAgents          []*Agent
	parents            []*Agent
	addDate            bool
	addEnvironmentInfo bool
	toolWrapper        toolWrapper
	memoryManager      memorymanager.Manager
}

// New creates a new agent
func New(name, prompt string, opts ...Opt) *Agent {
	agent := &Agent{
		name:        name,
		instruction: prompt,
	}

	for _, opt := range opts {
		opt(agent)
	}

	return agent
}

func (a *Agent) Name() string {
	return a.name
}

// Instruction returns the agent's instructions
func (a *Agent) Instruction() string {
	return a.instruction
}

func (a *Agent) AddDate() bool {
	return a.addDate
}

func (a *Agent) AddEnvironmentInfo() bool {
	return a.addEnvironmentInfo
}

// Description returns the agent's description
func (a *Agent) Description() string {
	return a.description
}

// SubAgents returns the list of sub-agent names
func (a *Agent) SubAgents() []*Agent {
	return a.subAgents
}

// Parents returns the list of parent agent names
func (a *Agent) Parents() []*Agent {
	return a.parents
}

// HasSubAgents checks if the agent has sub-agents
func (a *Agent) HasSubAgents() bool {
	return len(a.subAgents) > 0
}

func (a *Agent) HasParents() bool {
	return len(a.parents) > 0
}

// Model returns a random model from the available models
func (a *Agent) Model() provider.Provider {
	return a.models[rand.Intn(len(a.models))]
}

// ServerInfoProvider interface for toolsets that can provide server information
type ServerInfoProvider interface {
	GetServerInfo() (serverURL, serverType string)
}

// Tools returns the tools available to this agent
func (a *Agent) Tools(ctx context.Context) ([]tools.Tool, error) {
	if err := a.ensureToolSetsAreStarted(ctx); err != nil {
		// If this is an OAuth authorization error during startup, try to wrap it with server info
		if client.IsOAuthAuthorizationRequiredError(err) {
			// Try to find which toolset caused the OAuth error by checking each one
			for _, toolSet := range a.toolsets {
				if serverInfoProvider, ok := toolSet.(ServerInfoProvider); ok {
					serverURL, serverType := serverInfoProvider.GetServerInfo()
					return nil, &OAuthAuthorizationRequiredError{
						Err:        err,
						ServerURL:  serverURL,
						ServerType: serverType,
					}
				}
			}
		}
		return nil, err
	}

	var agentTools []tools.Tool
	for _, toolSet := range a.toolsets {
		ta, err := toolSet.Tools(ctx)
		if err != nil {
			return nil, err
		}
		agentTools = append(agentTools, ta...)
	}

	agentTools = append(agentTools, a.toolWrapper.allTools...)

	return agentTools, nil
}

func (a *Agent) ToolDisplayName(ctx context.Context, toolName string) string {
	allTools, err := a.Tools(ctx)
	if err != nil {
		slog.Error("Failed to get tools for display name", "agent", a.Name(), "error", err)
		return toolName
	}

	for _, tool := range allTools {
		if tool.Function.Name == toolName {
			return tool.DisplayName()
		}
	}

	return toolName
}

func (a *Agent) ToolSets() []tools.ToolSet {
	return a.toolsets
}

func (a *Agent) ensureToolSetsAreStarted(ctx context.Context) error {
	if a.toolsetsStarted.Load() {
		return nil
	}

	for _, toolSet := range a.toolsets {
		if err := toolSet.Start(ctx); err != nil {
			return fmt.Errorf("failed to start toolset: %w", err)
		}
	}

	a.toolsetsStarted.Store(true)
	return nil
}

func (a *Agent) StopToolSets() error {
	if !a.toolsetsStarted.Load() {
		return nil
	}

	for _, toolSet := range a.toolsets {
		if err := toolSet.Stop(); err != nil {
			return fmt.Errorf("failed to stop toolset: %w", err)
		}
	}

	a.toolsetsStarted.Store(false)
	return nil
}

package servicecore

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/docker/cagent/pkg/loader"
	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/session"
)

// Executor handles runtime creation and stream execution
type Executor struct {
	logger *slog.Logger
}

// NewExecutor creates a new runtime executor
func NewExecutor(logger *slog.Logger) *Executor {
	return &Executor{
		logger: logger,
	}
}

// CreateRuntime creates a new runtime instance for an agent
func (e *Executor) CreateRuntime(agentPath, agentName string, envFiles []string, gateway string) (*runtime.Runtime, *session.Session, error) {
	ctx := context.Background()
	
	e.logger.Debug("Creating runtime", "agent_path", agentPath, "agent_name", agentName)

	// Load agent configuration using existing loader
	agents, err := loader.Load(ctx, agentPath, envFiles, gateway, e.logger)
	if err != nil {
		return nil, nil, fmt.Errorf("loading agent configuration: %w", err)
	}

	// Start tool sets
	if err := agents.StartToolSets(ctx); err != nil {
		return nil, nil, fmt.Errorf("starting tool sets: %w", err)
	}

	// Create runtime
	rt, err := runtime.New(e.logger, agents, agentName)
	if err != nil {
		agents.StopToolSets() // Clean up on error
		return nil, nil, fmt.Errorf("creating runtime: %w", err)
	}

	// Create session
	sess := session.New(e.logger)

	e.logger.Debug("Runtime created successfully", "agent_name", agentName, "session_id", sess.ID)
	return rt, sess, nil
}

// ExecuteStream executes a message and collects streaming events into a response
func (e *Executor) ExecuteStream(rt *runtime.Runtime, sess *session.Session, message string) (*Response, error) {
	startTime := time.Now()
	
	e.logger.Debug("Executing stream", "session_id", sess.ID, "message_length", len(message))

	// Add user message to session
	sess.Messages = append(sess.Messages, session.UserMessage(message))

	// Start streaming execution
	ctx := context.Background()
	eventStream := rt.RunStream(ctx, sess)

	// Collect events and final response
	var events []runtime.Event
	var finalContent string
	toolCallCount := 0

	for event := range eventStream {
		events = append(events, event)
		
		switch evt := event.(type) {
		case *runtime.ToolCallEvent:
			toolCallCount++
			e.logger.Debug("Tool call event", "tool_name", evt.ToolCall.Function.Name)
			
		case *runtime.ToolCallResponseEvent:
			e.logger.Debug("Tool response event", "tool_name", evt.ToolCall.Function.Name, "response_length", len(evt.Response))
			
		case *runtime.AgentMessageEvent:
			finalContent = evt.Message.Content
			e.logger.Debug("Agent message event", "content_length", len(evt.Message.Content))
			
		case *runtime.ErrorEvent:
			e.logger.Error("Runtime error event", "error", evt.Error)
			return nil, fmt.Errorf("runtime execution error: %w", evt.Error)
		}
	}

	duration := time.Since(startTime)
	
	// Build response with metadata
	response := &Response{
		Content: finalContent,
		Events:  events,
		Metadata: map[string]interface{}{
			"duration_ms":    duration.Milliseconds(),
			"tool_calls":     toolCallCount,
			"event_count":    len(events),
			"session_id":     sess.ID,
			"message_length": len(message),
		},
	}

	e.logger.Debug("Stream execution completed", 
		"session_id", sess.ID, 
		"duration_ms", duration.Milliseconds(),
		"tool_calls", toolCallCount,
		"events", len(events))

	return response, nil
}

// CleanupRuntime cleans up runtime resources
func (e *Executor) CleanupRuntime(rt *runtime.Runtime) error {
	if rt == nil {
		return nil
	}

	// Stop tool sets
	if err := rt.Team().StopToolSets(); err != nil {
		e.logger.Warn("Error stopping tool sets during cleanup", "error", err)
		return fmt.Errorf("stopping tool sets: %w", err)
	}

	e.logger.Debug("Runtime cleaned up successfully")
	return nil
}
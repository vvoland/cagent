package types

import (
	"github.com/docker/cagent/pkg/tools"
)

// MessageType represents different types of messages
type MessageType int

const (
	MessageTypeUser MessageType = iota
	MessageTypeAssistant
	MessageTypeAssistantReasoning
	MessageTypeSpinner
	MessageTypeError
	MessageTypeShellOutput
	MessageTypeCancelled
	MessageTypeToolCall
	MessageTypeToolResult
	MessageTypeWelcome
)

// ToolStatus represents the status of a tool call
type ToolStatus int

const (
	ToolStatusPending ToolStatus = iota
	ToolStatusConfirmation
	ToolStatusRunning
	ToolStatusCompleted
	ToolStatusError
)

// Message represents a single message in the chat
type Message struct {
	Type           MessageType
	Content        string
	Sender         string         // Agent name for assistant messages
	ToolCall       tools.ToolCall // Associated tool call for tool messages
	ToolDefinition tools.Tool     // Definition of the tool being called
	ToolStatus     ToolStatus     // Status for tool calls
}

// Todo represents a single todo item
type Todo struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// TodoManager handles the shared state of all todos
// This is used by both the sidebar widget and tool components
type TodoManager struct {
	todos []Todo
}

// NewTodoManager creates a new TodoManager instance
func NewTodoManager() *TodoManager {
	return &TodoManager{
		todos: []Todo{},
	}
}

// AddTodo adds a new todo with the given id, description, and status
func (tm *TodoManager) AddTodo(id, description, status string) {
	tm.todos = append(tm.todos, Todo{
		ID:          id,
		Description: description,
		Status:      status,
	})
}

// UpdateTodo updates the status of a todo by id
// Returns true if the todo was found and updated, false otherwise
func (tm *TodoManager) UpdateTodo(id, status string) bool {
	for i, todo := range tm.todos {
		if todo.ID == id {
			tm.todos[i].Status = status
			return true
		}
	}
	return false
}

// GetTodos returns all todos
func (tm *TodoManager) GetTodos() []Todo {
	return tm.todos
}

// SessionState holds shared state across the TUI application.
// This provides a centralized location for state that needs to be
// accessible by multiple components.
type SessionState struct {
	// TodoManager manages the state of all todos across the application
	TodoManager *TodoManager

	// SplitDiffView determines whether diff views should be shown side-by-side (true)
	// or unified (false)
	SplitDiffView bool
}

// NewSessionState creates a new SessionState with default values.
func NewSessionState() *SessionState {
	return &SessionState{
		TodoManager:   NewTodoManager(),
		SplitDiffView: true, // Default to split view
	}
}

// ToggleSplitDiffView toggles between split and unified diff view modes.
func (s *SessionState) ToggleSplitDiffView() {
	s.SplitDiffView = !s.SplitDiffView
}

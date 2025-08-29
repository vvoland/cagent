package types

// MessageType represents different types of messages
type MessageType int

const (
	MessageTypeUser MessageType = iota
	MessageTypeAssistant
	MessageTypeSeparator
	MessageTypeToolCall
	MessageTypeToolResult
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
	Type       MessageType
	Content    string
	Sender     string     // Agent name for assistant messages
	ToolName   string     // Tool name for tool messages
	ToolCallID string     // Tool call ID for precise identification
	ToolStatus ToolStatus // Status for tool calls
	Arguments  string     // Arguments for tool calls
	Timestamp  int64
}

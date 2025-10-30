package commands

// Session commands
type (
	NewSessionMsg             struct{}
	EvalSessionMsg            struct{}
	CompactSessionMsg         struct{}
	CopySessionToClipboardMsg struct{}
)

// Agent commands
type AgentCommandMsg struct {
	Command string
}

package messages

// Session command messages
type (
	NewSessionMsg             struct{}
	ExitSessionMsg            struct{}
	EvalSessionMsg            struct{ Filename string }
	CompactSessionMsg         struct{ AdditionalPrompt string }
	CopySessionToClipboardMsg struct{}
	ExportSessionMsg          struct{ Filename string }
	ShowCostDialogMsg         struct{}
	ToggleYoloMsg             struct{}
	ToggleHideToolResultsMsg  struct{}
	StartShellMsg             struct{}
	SwitchAgentMsg            struct{ AgentName string }
	OpenSessionBrowserMsg     struct{}
	LoadSessionMsg            struct{ SessionID string }
	ToggleSessionStarMsg      struct{ SessionID string } // Toggle star on a session; empty ID means current session
	AttachFileMsg             struct{ FilePath string }  // Attach a file directly or open file picker if empty/directory
	InsertFileRefMsg          struct{ FilePath string }  // Insert @filepath reference into editor
)

// AgentCommandMsg command message
type AgentCommandMsg struct {
	Command string
}

// MCPPromptMsg command message
type MCPPromptMsg struct {
	PromptName string
	Arguments  map[string]string
}

// OpenURLMsg is a url for opening message
type OpenURLMsg struct {
	URL string
}

type ShowMCPPromptInputMsg struct {
	PromptName string
	PromptInfo any // mcptools.PromptInfo but avoiding import cycles
}

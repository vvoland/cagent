package tools

type ToolCall struct {
	Name string   `json:"name"`
	Args []string `json:"args"`
}

type ToolCallResult struct {
	Output string `json:"output"`
}

// OpenAI-like Tool Types

type ToolType string

type Tool struct {
	Type     ToolType            `json:"type"`
	Function *FunctionDefinition `json:"function,omitempty"`
}

type FunctionDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Strict      bool   `json:"strict,omitempty"`
	Parameters  any    `json:"parameters"`
}

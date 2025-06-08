package tools

import (
	"context"
)

type ToolHandler = func(ctx context.Context, toolCall ToolCall) (*ToolCallResult, error)

type ToolCall struct {
	Index    *int         `json:"index,omitempty"`
	ID       string       `json:"id,omitempty"`
	Type     ToolType     `json:"type"`
	Function FunctionCall `json:"function"`
}
type FunctionCall struct {
	Name string `json:"name,omitempty"`

	Arguments string `json:"arguments,omitempty"`
}

type ToolCallResult struct {
	Output string `json:"output"`
}

// OpenAI-like Tool Types

type ToolType string

type Tool struct {
	Function *FunctionDefinition `json:"function,omitempty"`
	Handler  ToolHandler         `json:"handler,omitempty"`
}

type FunctionDefinition struct {
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Strict      bool               `json:"strict,omitempty"`
	Parameters  FunctionParamaters `json:"parameters"`
}

type FunctionParamaters struct {
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties,omitempty"`
	Required   []string       `json:"required,omitempty"`
}

type ToolSet interface {
	Tools(ctx context.Context) ([]Tool, error)

	Instructions() string

	Start(ctx context.Context) error
	Stop() error
}

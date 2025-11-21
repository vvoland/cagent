package builtin

import (
	"context"

	"github.com/docker/cagent/pkg/tools"
)

const ToolNameHandoff = "handoff"

type HandoffTool struct {
	tools.ElicitationTool
}

// Make sure Handoff Tool implements the ToolSet Interface
var _ tools.ToolSet = (*HandoffTool)(nil)

type HandoffArgs struct {
	Agent string `json:"agent" jsonschema:"The name of the agent to hand off the conversation to."`
}

func NewHandoffTool() *HandoffTool {
	return &HandoffTool{}
}

func (t *HandoffTool) Instructions() string {
	return ""
}

func (t *HandoffTool) Tools(context.Context) ([]tools.Tool, error) {
	return []tools.Tool{
		{
			Name:        ToolNameHandoff,
			Category:    "handoff",
			Description: "Use this function to hand off the conversation to the selected agent.",
			Parameters:  tools.MustSchemaFor[HandoffArgs](),
			Annotations: tools.ToolAnnotations{
				ReadOnlyHint: true,
				Title:        "Handoff Conversation",
			},
		},
	}, nil
}

func (t *HandoffTool) Start(context.Context) error {
	return nil
}

func (t *HandoffTool) Stop(context.Context) error {
	return nil
}

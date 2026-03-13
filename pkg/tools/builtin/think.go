package builtin

import (
	"context"
	"strings"

	"github.com/docker/docker-agent/pkg/tools"
)

const ToolNameThink = "think"

type ThinkTool struct {
	thoughts []string
}

// Verify interface compliance
var (
	_ tools.ToolSet      = (*ThinkTool)(nil)
	_ tools.Instructable = (*ThinkTool)(nil)
)

type ThinkArgs struct {
	Thought string `json:"thought" jsonschema:"The thought to think about"`
}

func (t *ThinkTool) callTool(_ context.Context, params ThinkArgs) (*tools.ToolCallResult, error) {
	t.thoughts = append(t.thoughts, params.Thought)
	return tools.ResultSuccess("Thoughts:\n" + strings.Join(t.thoughts, "\n")), nil
}

func NewThinkTool() *ThinkTool {
	return &ThinkTool{}
}

func (t *ThinkTool) Instructions() string {
	return `## Think Tool

Use the think tool as a scratchpad before acting. Think to:
- Check which rules or policies apply
- Verify you have all required information
- Validate planned actions before executing
- Reason through complex multi-step problems`
}

func (t *ThinkTool) Tools(context.Context) ([]tools.Tool, error) {
	return []tools.Tool{
		{
			Name:         ToolNameThink,
			Category:     "think",
			Description:  "Use the tool to think about something. It will not obtain new information or change the database, but just append the thought to the log. Use it when complex reasoning or some cache memory is needed.",
			Parameters:   tools.MustSchemaFor[ThinkArgs](),
			OutputSchema: tools.MustSchemaFor[string](),
			Handler:      tools.NewHandler(t.callTool),
			Annotations: tools.ToolAnnotations{
				ReadOnlyHint: true,
				Title:        "Think",
			},
		},
	}, nil
}

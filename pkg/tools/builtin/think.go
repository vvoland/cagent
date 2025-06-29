package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rumpl/cagent/pkg/tools"
)

type ThinkTool struct {
	handler *thinkHandler
}

type thinkHandler struct {
	thoughts []string
}

func (h *thinkHandler) CallTool(ctx context.Context, toolCall tools.ToolCall) (*tools.ToolCallResult, error) {
	var params struct {
		Thought string `json:"thought"`
	}

	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	h.thoughts = append(h.thoughts, params.Thought)
	return &tools.ToolCallResult{
		Output: "Thoughts:\n" + strings.Join(h.thoughts, "\n"),
	}, nil
}

func NewThinkTool() *ThinkTool {
	return &ThinkTool{
		handler: &thinkHandler{},
	}
}

func (t *ThinkTool) Instructions() string {
	return `## Using the think tool
            Before taking any action or responding to the user after receiving tool results, use the think tool as a scratchpad to:
            - List the specific rules that apply to the current request
            - Check if all required information is collected
            - Verify that the planned action complies with all policies
            - Iterate over tool results for correctness

            ## Rules
            - Use the think tool generously to jot down thoughts and ideas.`
}

func (t *ThinkTool) Tools(ctx context.Context) ([]tools.Tool, error) {
	return []tools.Tool{
		{
			Function: &tools.FunctionDefinition{
				Name:        "think",
				Description: "Use the tool to think about something. It will not obtain new information or change the database, but just append the thought to the log. Use it when complex reasoning or some cache memory is needed.",
				Parameters: tools.FunctionParamaters{
					Type: "object",
					Properties: map[string]any{
						"thought": map[string]any{
							"type":        "string",
							"description": "The thought to think about",
						},
					},
					Required: []string{"thought"},
				},
			},
			Handler: t.handler.CallTool,
		},
	}, nil
}

func (t *ThinkTool) Start(ctx context.Context) error {
	return nil
}

func (t *ThinkTool) Stop() error {
	return nil
}

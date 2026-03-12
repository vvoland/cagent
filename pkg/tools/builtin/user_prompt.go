package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/docker/docker-agent/pkg/tools"
)

const ToolNameUserPrompt = "user_prompt"

type UserPromptTool struct {
	elicitationHandler tools.ElicitationHandler
}

// Verify interface compliance
var (
	_ tools.ToolSet      = (*UserPromptTool)(nil)
	_ tools.Elicitable   = (*UserPromptTool)(nil)
	_ tools.Instructable = (*UserPromptTool)(nil)
)

// UserPromptOption represents a single selectable choice presented to the user.
type UserPromptOption struct {
	Label       string `json:"label" jsonschema:"Short display text for this option (1-5 words)"`
	Description string `json:"description" jsonschema:"Brief explanation of what this option means"`
}

type UserPromptArgs struct {
	Message  string             `json:"message" jsonschema:"The message/question to display to the user"`
	Title    string             `json:"title,omitempty" jsonschema:"Optional title for the dialog window (defaults to 'Question')"`
	Schema   map[string]any     `json:"schema,omitempty" jsonschema:"JSON Schema defining the expected response structure. Mutually exclusive with options."`
	Options  []UserPromptOption `json:"options,omitempty" jsonschema:"List of choices to present to the user. Each has a label and description. The user can pick from these or type a custom answer. Put recommended option first and append '(Recommended)' to its label. Mutually exclusive with schema."`
	Multiple bool               `json:"multiple,omitempty" jsonschema:"When true and options are provided, allow the user to select multiple options. Defaults to single selection."`
}

type UserPromptResponse struct {
	Action  string         `json:"action" jsonschema:"The user action: accept, decline, or cancel"`
	Content map[string]any `json:"content,omitempty" jsonschema:"The user response data (only present when action is accept)"`
}

func NewUserPromptTool() *UserPromptTool {
	return &UserPromptTool{}
}

func (t *UserPromptTool) SetElicitationHandler(handler tools.ElicitationHandler) {
	t.elicitationHandler = handler
}

func (t *UserPromptTool) userPrompt(ctx context.Context, params UserPromptArgs) (*tools.ToolCallResult, error) {
	if t.elicitationHandler == nil {
		return tools.ResultError("user_prompt tool is not available in this context (no elicitation handler configured)"), nil
	}

	meta := mcp.Meta{}
	if params.Title != "" {
		meta["cagent/title"] = params.Title
	}
	if len(params.Options) > 0 {
		meta["cagent/options"] = params.Options
		meta["cagent/multiple"] = params.Multiple
	}

	req := &mcp.ElicitParams{
		Message:         params.Message,
		RequestedSchema: params.Schema,
		Meta:            meta,
	}

	result, err := t.elicitationHandler(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("elicitation request failed: %w", err)
	}

	response := UserPromptResponse{
		Action:  string(result.Action),
		Content: result.Content,
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	if result.Action != tools.ElicitationActionAccept {
		return tools.ResultError(string(responseJSON)), nil
	}

	return tools.ResultSuccess(string(responseJSON)), nil
}

func (t *UserPromptTool) Instructions() string {
	return `## Using the user_prompt tool

Use user_prompt to ask the user a question or gather input when you need clarification, specific information, or a decision.

Optionally provide a "title" to label the dialog (defaults to "Question").

### Presenting choices with options (preferred for decisions)

Provide "options" — a list of {label, description} objects — to present clickable choices.
The user can select from the list or type a custom answer.
- Put the recommended option first and append "(Recommended)" to its label.
- Set "multiple": true to allow selecting more than one option.
- Do NOT include catch-all options like "Other" — a custom text input is always available.

Example with options:
{"message": "Which base image strategy?", "options": [{"label": "Alpine multi-stage (Recommended)", "description": "Smallest image size, widely used"}, {"label": "Distroless runtime", "description": "No shell, minimal attack surface"}, {"label": "Scratch with static binary", "description": "Absolute minimum, requires CGO_ENABLED=0"}]}

### Structured input with schema (for forms)

Provide a JSON "schema" to collect structured data (object, primitive, or enum types).
If neither options nor schema is provided, the user can type a free-form response.

Example schema for structured input:
{"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]}

### Response format

Response contains "action" (accept/decline/cancel) and "content" (user data, only when accepted).
When options are used, content has "selection" (array of selected labels) or "custom" (user-typed text).`
}

func (t *UserPromptTool) Tools(context.Context) ([]tools.Tool, error) {
	return []tools.Tool{
		{
			Name:         ToolNameUserPrompt,
			Category:     "user_prompt",
			Description:  "Ask the user a question and wait for their response. Use this when you need interactive input, clarification, or confirmation from the user. Provide 'options' to present a list of choices, or a JSON 'schema' for structured input.",
			Parameters:   tools.MustSchemaFor[UserPromptArgs](),
			OutputSchema: tools.MustSchemaFor[UserPromptResponse](),
			Handler:      tools.NewHandler(t.userPrompt),
			Annotations: tools.ToolAnnotations{
				ReadOnlyHint: true,
				Title:        "User Prompt",
			},
		},
	}, nil
}

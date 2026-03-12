package builtin

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker-agent/pkg/tools"
)

const (
	ToolNameChangeModel = "change_model"
	ToolNameRevertModel = "revert_model"
)

// ModelPickerTool provides tools for dynamically switching the agent's model mid-conversation.
type ModelPickerTool struct {
	models []string // list of available model references
}

// Verify interface compliance
var (
	_ tools.ToolSet      = (*ModelPickerTool)(nil)
	_ tools.Instructable = (*ModelPickerTool)(nil)
)

// ChangeModelArgs are the arguments for the change_model tool.
type ChangeModelArgs struct {
	Model string `json:"model" jsonschema:"The model to switch to. Must be one of the available models."`
}

// NewModelPickerTool creates a new ModelPickerTool with the given list of allowed models.
func NewModelPickerTool(models []string) *ModelPickerTool {
	return &ModelPickerTool{models: models}
}

// Instructions returns guidance for the LLM on when and how to use the model picker tools.
func (t *ModelPickerTool) Instructions() string {
	return "## Model Switching\n\n" +
		"Available models: " + strings.Join(t.models, ", ") + ".\n\n" +
		"Use `" + ToolNameChangeModel + "` to switch to a model better suited for the current task " +
		"(e.g., faster model for simple tasks, more capable model for complex reasoning).\n" +
		"Use `" + ToolNameRevertModel + "` to return to the original model when done."
}

// AllowedModels returns the list of models this tool allows switching to.
func (t *ModelPickerTool) AllowedModels() []string {
	return t.models
}

// Tools returns the change_model and revert_model tool definitions.
func (t *ModelPickerTool) Tools(context.Context) ([]tools.Tool, error) {
	return []tools.Tool{
		{
			Name:     ToolNameChangeModel,
			Category: "model",
			Description: fmt.Sprintf(
				"Change the current model to one of the available models: %s. "+
					"Use this when you need a different model for the current task.",
				strings.Join(t.models, ", "),
			),
			Parameters: tools.MustSchemaFor[ChangeModelArgs](),
			Annotations: tools.ToolAnnotations{
				ReadOnlyHint: true,
				Title:        "Change Model",
			},
		},
		{
			Name:     ToolNameRevertModel,
			Category: "model",
			Description: "Revert to the agent's original/default model. " +
				"Use this after completing a task that required a different model.",
			Annotations: tools.ToolAnnotations{
				ReadOnlyHint: true,
				Title:        "Revert Model",
			},
		},
	}, nil
}

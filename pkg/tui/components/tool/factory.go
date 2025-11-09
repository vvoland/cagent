package tool

import (
	"github.com/charmbracelet/glamour/v2"

	"github.com/docker/cagent/pkg/app"
	"github.com/docker/cagent/pkg/tools/builtin"
	"github.com/docker/cagent/pkg/tui/components/tool/defaulttool"
	"github.com/docker/cagent/pkg/tui/components/tool/editfile"
	"github.com/docker/cagent/pkg/tui/components/tool/todotool"
	"github.com/docker/cagent/pkg/tui/components/tool/transfertask"
	"github.com/docker/cagent/pkg/tui/core/layout"
	"github.com/docker/cagent/pkg/tui/types"
)

// Factory creates tool components using the registry.
// It looks up registered component builders and falls back to a default component
// if no specific builder is registered for a tool.
type Factory struct {
	registry *Registry
}

// NewFactory creates a new component factory with the given registry.
func NewFactory(registry *Registry) *Factory {
	return &Factory{
		registry: registry,
	}
}

// Create creates a tool component for the given message.
func (f *Factory) Create(
	msg *types.Message,
	a *app.App,
	renderer *glamour.TermRenderer,
	sessionState *types.SessionState,
) layout.Model {
	toolName := msg.ToolCall.Function.Name

	if builder, ok := f.registry.Get(toolName); ok {
		return builder(msg, a, renderer, sessionState)
	}

	return defaulttool.New(msg, a, renderer, sessionState)
}

var (
	defaultRegistry = newDefaultRegistry()
	defaultFactory  = NewFactory(defaultRegistry)
)

func newDefaultRegistry() *Registry {
	registry := NewRegistry()

	registry.Register("edit_file", editfile.New)
	registry.Register("transfer_task", transfertask.New)

	// Register unified todo tool component for all todo operations
	registry.Register(builtin.ToolNameCreateTodo, todotool.New)
	registry.Register(builtin.ToolNameCreateTodos, todotool.New)
	registry.Register(builtin.ToolNameUpdateTodo, todotool.New)
	registry.Register(builtin.ToolNameListTodos, todotool.New)

	return registry
}

// New creates a tool component using the default factory.
// This is the public API that maintains backward compatibility.
func New(
	msg *types.Message,
	a *app.App,
	renderer *glamour.TermRenderer,
	sessionState *types.SessionState,
) layout.Model {
	return defaultFactory.Create(msg, a, renderer, sessionState)
}

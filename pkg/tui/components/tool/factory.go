package tool

import (
	"github.com/docker/cagent/pkg/tools/builtin"
	"github.com/docker/cagent/pkg/tui/components/tool/allowed"
	"github.com/docker/cagent/pkg/tui/components/tool/api"
	"github.com/docker/cagent/pkg/tui/components/tool/defaulttool"
	"github.com/docker/cagent/pkg/tui/components/tool/editfile"
	"github.com/docker/cagent/pkg/tui/components/tool/handoff"
	"github.com/docker/cagent/pkg/tui/components/tool/listdirectory"
	"github.com/docker/cagent/pkg/tui/components/tool/readfile"
	"github.com/docker/cagent/pkg/tui/components/tool/readmultiplefiles"
	"github.com/docker/cagent/pkg/tui/components/tool/searchfiles"
	"github.com/docker/cagent/pkg/tui/components/tool/shell"
	"github.com/docker/cagent/pkg/tui/components/tool/todotool"
	"github.com/docker/cagent/pkg/tui/components/tool/transfertask"
	"github.com/docker/cagent/pkg/tui/components/tool/writefile"
	"github.com/docker/cagent/pkg/tui/core/layout"
	"github.com/docker/cagent/pkg/tui/service"
	"github.com/docker/cagent/pkg/tui/types"
)

// Factory creates tool components using the registry.
// It looks up registered component builders and falls back to a default component
// if no specific builder is registered for a tool.
type Factory struct {
	registry *Registry
}

func NewFactory(registry *Registry) *Factory {
	return &Factory{
		registry: registry,
	}
}

func (f *Factory) Create(msg *types.Message, sessionState *service.SessionState) layout.Model {
	toolName := msg.ToolCall.Function.Name

	// First try to match by exact tool name
	if builder, ok := f.registry.Get(toolName); ok {
		return builder(msg, sessionState)
	}

	// Then try to match by category
	if msg.ToolDefinition.Category != "" {
		if builder, ok := f.registry.Get("category:" + msg.ToolDefinition.Category); ok {
			return builder(msg, sessionState)
		}
	}

	return defaulttool.New(msg, sessionState)
}

var (
	defaultRegistry = newDefaultRegistry()
	defaultFactory  = NewFactory(defaultRegistry)
)

// toolRegistration pairs tool names with their component builder.
type toolRegistration struct {
	names   []string
	builder ComponentBuilder
}

func newDefaultRegistry() *Registry {
	registry := NewRegistry()

	// Define tool registrations in a declarative manner.
	// Tools with the same builder are grouped together.
	registrations := []toolRegistration{
		{[]string{builtin.ToolNameTransferTask}, transfertask.New},
		{[]string{builtin.ToolNameHandoff}, handoff.New},
		{[]string{builtin.ToolNameEditFile}, editfile.New},
		{[]string{builtin.ToolNameWriteFile}, writefile.New},
		{[]string{builtin.ToolNameReadFile}, readfile.New},
		{[]string{builtin.ToolNameReadMultipleFiles}, readmultiplefiles.New},
		{[]string{builtin.ToolNameListDirectory}, listdirectory.New},
		{[]string{builtin.ToolNameSearchFiles}, searchfiles.New},
		{[]string{builtin.ToolNameShell}, shell.New},
		{[]string{builtin.ToolNameAddAllowedDirectory}, allowed.New},
		{[]string{builtin.ToolNameFetch, "category:api"}, api.New},
		{
			[]string{
				builtin.ToolNameCreateTodo,
				builtin.ToolNameCreateTodos,
				builtin.ToolNameUpdateTodo,
				builtin.ToolNameListTodos,
			},
			todotool.New,
		},
	}

	for _, reg := range registrations {
		for _, name := range reg.names {
			registry.Register(name, reg.builder)
		}
	}

	return registry
}

func New(msg *types.Message, sessionState *service.SessionState) layout.Model {
	return defaultFactory.Create(msg, sessionState)
}

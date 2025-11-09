package tool

import (
	"sync"

	"github.com/charmbracelet/glamour/v2"

	"github.com/docker/cagent/pkg/app"
	"github.com/docker/cagent/pkg/tui/core/layout"
	"github.com/docker/cagent/pkg/tui/service"
	"github.com/docker/cagent/pkg/tui/types"
)

// ComponentBuilder is a function that creates a tool component.
type ComponentBuilder func(
	msg *types.Message,
	app *app.App,
	renderer *glamour.TermRenderer,
	sessionState *service.SessionState,
) layout.Model

// Registry manages tool component builders.
type Registry struct {
	mu       sync.RWMutex
	builders map[string]ComponentBuilder
}

// NewRegistry creates a new component registry.
func NewRegistry() *Registry {
	return &Registry{
		builders: make(map[string]ComponentBuilder),
	}
}

// Register adds a component builder for a tool name.
// If a builder already exists for this tool name, it will be replaced.
func (r *Registry) Register(toolName string, builder ComponentBuilder) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.builders[toolName] = builder
}

// Get retrieves a component builder for a tool name.
// Returns the builder and true if found, nil and false otherwise.
func (r *Registry) Get(toolName string) (ComponentBuilder, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	builder, exists := r.builders[toolName]
	return builder, exists
}

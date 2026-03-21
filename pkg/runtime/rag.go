package runtime

import "context"

// StartBackgroundRAGInit is a no-op. RAG initialization is now handled
// per-toolset via the tools.Startable interface.
func (r *LocalRuntime) StartBackgroundRAGInit(_ context.Context, _ func(Event)) {
	// RAG toolsets are initialized lazily when first used.
}

// InitializeRAG is a no-op. RAG initialization is now handled
// per-toolset via the tools.Startable interface.
func (r *LocalRuntime) InitializeRAG(_ context.Context, _ chan Event) {
	// RAG toolsets are initialized lazily when first used.
}

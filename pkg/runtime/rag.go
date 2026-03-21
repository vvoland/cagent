package runtime

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/docker/docker-agent/pkg/rag"
	"github.com/docker/docker-agent/pkg/rag/types"
)

// StartBackgroundRAGInit initializes RAG in background and forwards events
// Should be called early (e.g., by App) to start indexing before RunStream
func (r *LocalRuntime) StartBackgroundRAGInit(ctx context.Context, sendEvent func(Event)) {
	if r.ragInitialized.Swap(true) {
		return
	}

	ragManagers := r.team.RAGManagers()
	if len(ragManagers) == 0 {
		return
	}

	// Set up event forwarding BEFORE starting initialization
	r.forwardRAGEvents(ctx, ragManagers, sendEvent)
	initializeRAG(ctx, ragManagers)
	startRAGFileWatchers(ctx, ragManagers)
}

// forwardRAGEvents forwards RAG manager events to the given callback
// Consolidates duplicated event forwarding logic
func (r *LocalRuntime) forwardRAGEvents(ctx context.Context, ragManagers []*rag.Manager, sendEvent func(Event)) {
	for _, mgr := range ragManagers {
		go func() {
			ragName := mgr.Name()
			slog.Debug("Starting RAG event forwarder goroutine", "rag", ragName)
			for {
				select {
				case <-ctx.Done():
					slog.Debug("RAG event forwarder stopped", "rag", ragName)
					return
				case ragEvent, ok := <-mgr.Events():
					if !ok {
						slog.Debug("RAG events channel closed", "rag", ragName)
						return
					}

					agentName := r.CurrentAgentName()
					slog.Debug("Forwarding RAG event", "type", ragEvent.Type, "rag", ragName, "agent", agentName)

					switch ragEvent.Type {
					case types.EventTypeIndexingStarted:
						sendEvent(RAGIndexingStarted(ragName, ragEvent.StrategyName))
					case types.EventTypeIndexingProgress:
						if ragEvent.Progress != nil {
							sendEvent(RAGIndexingProgress(ragName, ragEvent.StrategyName, ragEvent.Progress.Current, ragEvent.Progress.Total, agentName))
						}
					case types.EventTypeIndexingComplete:
						sendEvent(RAGIndexingCompleted(ragName, ragEvent.StrategyName))
					case types.EventTypeUsage:
						// Convert RAG usage to TokenUsageEvent so TUI displays it
						sendEvent(NewTokenUsageEvent("", agentName, &Usage{
							InputTokens:   ragEvent.TotalTokens,
							ContextLength: ragEvent.TotalTokens,
							Cost:          ragEvent.Cost,
						}))
					case types.EventTypeError:
						if ragEvent.Error != nil {
							sendEvent(Error(fmt.Sprintf("RAG %s error: %v", ragName, ragEvent.Error)))
						}
					default:
						// Log unhandled events for debugging
						slog.Debug("Unhandled RAG event type", "type", ragEvent.Type, "rag", ragName)
					}
				}
			}
		}()
	}
}

// InitializeRAG initializes all RAG managers in the background
func initializeRAG(ctx context.Context, ragManagers []*rag.Manager) {
	for _, mgr := range ragManagers {
		go func() {
			slog.Debug("Starting RAG manager initialization goroutine", "rag", mgr.Name())
			if err := mgr.Initialize(ctx); err != nil {
				slog.Error("Failed to initialize RAG manager", "rag", mgr.Name(), "error", err)
			} else {
				slog.Info("RAG manager initialized successfully", "rag", mgr.Name())
			}
		}()
	}
}

// StartRAGFileWatchers starts file watchers for all RAG managers
func startRAGFileWatchers(ctx context.Context, ragManagers []*rag.Manager) {
	for _, mgr := range ragManagers {
		go func() {
			slog.Debug("Starting RAG file watcher goroutine", "rag", mgr.Name())
			if err := mgr.StartFileWatcher(ctx); err != nil {
				slog.Error("Failed to start RAG file watcher", "rag", mgr.Name(), "error", err)
			}
		}()
	}
}

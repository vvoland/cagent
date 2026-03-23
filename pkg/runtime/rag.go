package runtime

import (
	"context"
	"fmt"
	"log/slog"

	ragtypes "github.com/docker/docker-agent/pkg/rag/types"
	"github.com/docker/docker-agent/pkg/tools"
	"github.com/docker/docker-agent/pkg/tools/builtin"
)

// StartBackgroundRAGInit discovers RAG toolsets from agents and wires up event
// forwarding so the TUI can display indexing progress. Actual initialization
// happens lazily when the tool is first used (via tools.Startable).
func (r *LocalRuntime) StartBackgroundRAGInit(ctx context.Context, sendEvent func(Event)) {
	for _, name := range r.team.AgentNames() {
		a, err := r.team.Agent(name)
		if err != nil {
			continue
		}
		for _, ts := range a.ToolSets() {
			ragTool, ok := tools.As[*builtin.RAGTool](ts)
			if !ok {
				continue
			}
			ragTool.SetEventCallback(ragEventForwarder(ctx, ragTool.Name(), r, sendEvent))
		}
	}
}

// ragEventForwarder returns a callback that converts RAG manager events to runtime events.
func ragEventForwarder(ctx context.Context, ragName string, r *LocalRuntime, sendEvent func(Event)) builtin.RAGEventCallback {
	return func(ragEvent ragtypes.Event) {
		agentName := r.CurrentAgentName()
		slog.Debug("Forwarding RAG event", "type", ragEvent.Type, "rag", ragName, "agent", agentName)

		switch ragEvent.Type {
		case ragtypes.EventTypeIndexingStarted:
			sendEvent(RAGIndexingStarted(ragName, ragEvent.StrategyName))
		case ragtypes.EventTypeIndexingProgress:
			if ragEvent.Progress != nil {
				sendEvent(RAGIndexingProgress(ragName, ragEvent.StrategyName, ragEvent.Progress.Current, ragEvent.Progress.Total, agentName))
			}
		case ragtypes.EventTypeIndexingComplete:
			sendEvent(RAGIndexingCompleted(ragName, ragEvent.StrategyName))
		case ragtypes.EventTypeUsage:
			sendEvent(NewTokenUsageEvent("", agentName, &Usage{
				InputTokens:   ragEvent.TotalTokens,
				ContextLength: ragEvent.TotalTokens,
				Cost:          ragEvent.Cost,
			}))
		case ragtypes.EventTypeError:
			if ragEvent.Error != nil {
				sendEvent(Error(fmt.Sprintf("RAG %s error: %v", ragName, ragEvent.Error)))
			}
		default:
			slog.Debug("Unhandled RAG event type", "type", ragEvent.Type, "rag", ragName)
		}

		_ = ctx // available for future use
	}
}

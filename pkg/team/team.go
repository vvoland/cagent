package team

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/docker/cagent/pkg/agent"
	"github.com/docker/cagent/pkg/rag"
)

type Team struct {
	ID          string
	agents      map[string]*agent.Agent
	ragManagers map[string]*rag.Manager
}

type Opt func(*Team)

func WithID(id string) Opt {
	return func(t *Team) {
		t.ID = id
	}
}

func WithAgents(agents ...*agent.Agent) Opt {
	return func(t *Team) {
		for _, agent := range agents {
			t.agents[agent.Name()] = agent
		}
	}
}

func WithRAGManagers(managers map[string]*rag.Manager) Opt {
	return func(t *Team) {
		t.ragManagers = managers
	}
}

func New(opts ...Opt) *Team {
	t := &Team{
		agents:      make(map[string]*agent.Agent),
		ragManagers: make(map[string]*rag.Manager),
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

func (t *Team) AgentNames() []string {
	var names []string
	for name := range t.agents {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func (t *Team) Agent(name string) (*agent.Agent, error) {
	if t.Size() == 0 {
		return nil, errors.New("no agents loaded; ensure your agent configuration defines at least one agent")
	}

	found, ok := t.agents[name]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s (available agents: %s)", name, strings.Join(t.AgentNames(), ", "))
	}

	return found, nil
}

func (t *Team) Size() int {
	return len(t.agents)
}

func (t *Team) StopToolSets(ctx context.Context) error {
	for _, agent := range t.agents {
		if err := agent.StopToolSets(ctx); err != nil {
			return fmt.Errorf("failed to stop tool sets: %w", err)
		}
	}
	for name, mgr := range t.ragManagers {
		if err := mgr.Close(); err != nil {
			slog.Error("Failed to close RAG manager", "name", name, "error", err)
		}
	}

	return nil
}

// RAGManagers returns the RAG managers for this team
func (t *Team) RAGManagers() map[string]*rag.Manager {
	return t.ragManagers
}

// InitializeRAG initializes all RAG managers in the background
func (t *Team) InitializeRAG(ctx context.Context) {
	for _, mgr := range t.ragManagers {
		go func(m *rag.Manager) {
			slog.Debug("Starting RAG manager initialization goroutine", "rag", m.Name())
			if err := m.Initialize(ctx); err != nil {
				slog.Error("Failed to initialize RAG manager", "rag", m.Name(), "error", err)
			} else {
				slog.Info("RAG manager initialized successfully", "rag", m.Name())
			}
		}(mgr)
	}
}

// StartRAGFileWatchers starts file watchers for all RAG managers
func (t *Team) StartRAGFileWatchers(ctx context.Context) {
	for _, mgr := range t.ragManagers {
		go func(m *rag.Manager) {
			slog.Debug("Starting RAG file watcher goroutine", "rag", m.Name())
			if err := m.StartFileWatcher(ctx); err != nil {
				slog.Error("Failed to start RAG file watcher", "rag", m.Name(), "error", err)
			}
		}(mgr)
	}
}

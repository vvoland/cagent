package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/docker/cagent/pkg/concurrent"
	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/team"
	"github.com/docker/cagent/pkg/teamloader"
)

type sessionManager struct {
	runtimes *concurrent.Map[string, runtime.Runtime]
	sources  config.Sources

	refreshInterval time.Duration
}

func newSessionManager(sources config.Sources, refreshInterval time.Duration) *sessionManager {
	loaders := make(config.Sources)
	for name, source := range sources {
		loaders[name] = newSourceLoader(source, refreshInterval)
	}

	sm := &sessionManager{
		runtimes:        concurrent.NewMap[string, runtime.Runtime](),
		sources:         loaders,
		refreshInterval: refreshInterval,
	}

	return sm
}

func (sm *sessionManager) runtimeForSession(ctx context.Context, sess *session.Session, agentFilename, currentAgent string, rc *config.RuntimeConfig) (runtime.Runtime, error) {
	rt, exists := sm.runtimes.Load(sess.ID)
	if exists {
		return rt, nil
	}

	t, err := sm.loadTeam(ctx, agentFilename, rc)
	if err != nil {
		return nil, err
	}

	agent, err := t.Agent(currentAgent)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("agent not found: %v", err))
	}
	sess.MaxIterations = agent.MaxIterations()

	opts := []runtime.Opt{
		runtime.WithCurrentAgent(currentAgent),
		runtime.WithManagedOAuth(false),
		runtime.WithRootSessionID(sess.ID),
	}
	rt, err = runtime.New(t, opts...)
	if err != nil {
		slog.Error("Failed to create runtime", "error", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to create runtime: %v", err))
	}
	sm.runtimes.Store(sess.ID, rt)
	slog.Debug("Runtime created for session", "session_id", sess.ID)

	return rt, nil
}

func (sm *sessionManager) loadTeam(ctx context.Context, agentFilename string, runConfig *config.RuntimeConfig) (*team.Team, error) {
	agentSource, found := sm.sources[agentFilename]
	if !found {
		return nil, fmt.Errorf("agent not found: %s", agentFilename)
	}

	return teamloader.Load(ctx, agentSource, runConfig)
}

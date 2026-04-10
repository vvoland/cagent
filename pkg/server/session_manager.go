package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker-agent/pkg/api"
	"github.com/docker/docker-agent/pkg/concurrent"
	"github.com/docker/docker-agent/pkg/config"
	"github.com/docker/docker-agent/pkg/runtime"
	"github.com/docker/docker-agent/pkg/session"
	"github.com/docker/docker-agent/pkg/sessiontitle"
	"github.com/docker/docker-agent/pkg/team"
	"github.com/docker/docker-agent/pkg/teamloader"
	"github.com/docker/docker-agent/pkg/tools"
)

type activeRuntimes struct {
	runtime   runtime.Runtime
	cancel    context.CancelFunc
	session   *session.Session        // The actual session object used by the runtime
	titleGen  *sessiontitle.Generator // Title generator (includes fallback models)
	streaming bool                    // True while RunStream is active; prevents concurrent runs
}

// SessionManager manages sessions for HTTP and Connect-RPC servers.
type SessionManager struct {
	runtimeSessions *concurrent.Map[string, *activeRuntimes]
	sessionStore    session.Store
	Sources         config.Sources

	// TODO: We have to do something about this, it's weird, session creation should send everything that is needed.
	// This is only used for the working directory...
	runConfig *config.RuntimeConfig

	refreshInterval time.Duration

	mux sync.Mutex
}

// NewSessionManager creates a new session manager.
func NewSessionManager(ctx context.Context, sources config.Sources, sessionStore session.Store, refreshInterval time.Duration, runConfig *config.RuntimeConfig) *SessionManager {
	loaders := make(config.Sources)
	for name, source := range sources {
		loaders[name] = newSourceLoader(ctx, source, refreshInterval)
	}

	sm := &SessionManager{
		runtimeSessions: concurrent.NewMap[string, *activeRuntimes](),
		sessionStore:    sessionStore,
		Sources:         loaders,
		refreshInterval: refreshInterval,
		runConfig:       runConfig,
	}

	return sm
}

// GetSession retrieves a session by ID.
func (sm *SessionManager) GetSession(ctx context.Context, id string) (*session.Session, error) {
	sess, err := sm.sessionStore.GetSession(ctx, id)
	if err != nil {
		return nil, err
	}
	return sess, nil
}

// CreateSession creates a new session from a template.
func (sm *SessionManager) CreateSession(ctx context.Context, sessionTemplate *session.Session) (*session.Session, error) {
	var opts []session.Opt
	opts = append(opts,
		session.WithMaxIterations(sessionTemplate.MaxIterations),
		session.WithMaxConsecutiveToolCalls(sessionTemplate.MaxConsecutiveToolCalls),
		session.WithMaxOldToolCallTokens(sessionTemplate.MaxOldToolCallTokens),
		session.WithToolsApproved(sessionTemplate.ToolsApproved),
	)

	if wd := strings.TrimSpace(sessionTemplate.WorkingDir); wd != "" {
		absWd, err := filepath.Abs(wd)
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(absWd)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			return nil, errors.New("working directory must be a directory")
		}
		opts = append(opts, session.WithWorkingDir(absWd))
	}

	if sessionTemplate.Permissions != nil {
		opts = append(opts, session.WithPermissions(sessionTemplate.Permissions))
	}

	sess := session.New(opts...)
	return sess, sm.sessionStore.AddSession(ctx, sess)
}

// GetSessions retrieves all sessions.
func (sm *SessionManager) GetSessions(ctx context.Context) ([]*session.Session, error) {
	sessions, err := sm.sessionStore.GetSessions(ctx)
	if err != nil {
		return nil, err
	}
	return sessions, nil
}

// DeleteSession deletes a session by ID.
func (sm *SessionManager) DeleteSession(ctx context.Context, sessionID string) error {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	sess, err := sm.sessionStore.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	if err := sm.sessionStore.DeleteSession(ctx, sessionID); err != nil {
		return err
	}

	if sessionRuntime, ok := sm.runtimeSessions.Load(sess.ID); ok {
		sessionRuntime.cancel()
		sm.runtimeSessions.Delete(sess.ID)
	}

	return nil
}

// RunSession runs a session with the given messages.
func (sm *SessionManager) RunSession(ctx context.Context, sessionID, agentFilename, currentAgent string, messages []api.Message) (<-chan runtime.Event, error) {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	sess, err := sm.sessionStore.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	rc := sm.runConfig.Clone()
	rc.WorkingDir = sess.WorkingDir

	// Collect user messages for potential title generation
	var userMessages []string
	for _, msg := range messages {
		sess.AddMessage(session.UserMessage(msg.Content, msg.MultiContent...))
		if msg.Content != "" {
			userMessages = append(userMessages, msg.Content)
		}
	}

	if err := sm.sessionStore.UpdateSession(ctx, sess); err != nil {
		return nil, err
	}

	runtimeSession, exists := sm.runtimeSessions.Load(sessionID)

	// Reject if a stream is already active for this session. The caller
	// should use POST /sessions/:id/steer to inject follow-up messages
	// into a running session instead of starting a second concurrent stream.
	if exists && runtimeSession.streaming {
		return nil, errors.New("session is already streaming; use /steer to send follow-up messages")
	}

	streamCtx, cancel := context.WithCancel(ctx)
	var titleGen *sessiontitle.Generator
	if !exists {
		var rt runtime.Runtime
		rt, titleGen, err = sm.runtimeForSession(ctx, sess, agentFilename, currentAgent, rc)
		if err != nil {
			cancel()
			return nil, err
		}
		runtimeSession = &activeRuntimes{
			runtime:  rt,
			cancel:   cancel,
			session:  sess,
			titleGen: titleGen,
		}
		sm.runtimeSessions.Store(sessionID, runtimeSession)
	} else {
		// Update the session pointer in case it was reloaded
		runtimeSession.session = sess
		titleGen = runtimeSession.titleGen
	}

	runtimeSession.streaming = true

	streamChan := make(chan runtime.Event)

	// Check if we need to generate a title
	needsTitle := sess.Title == "" && len(userMessages) > 0 && titleGen != nil

	go func() {
		// Start title generation in parallel if needed
		if needsTitle {
			go sm.generateTitle(ctx, sess, titleGen, userMessages, streamChan)
		}

		stream := runtimeSession.runtime.RunStream(streamCtx, sess)
		// Single defer to control ordering: clear the streaming flag
		// BEFORE closing streamChan. When the client sees the channel
		// close it may immediately call RunSession for the next queued
		// message; streaming must already be false by then.
		defer func() {
			sm.mux.Lock()
			runtimeSession.streaming = false
			sm.mux.Unlock()
			close(streamChan)
			cancel()
		}()
		for event := range stream {
			if streamCtx.Err() != nil {
				return
			}
			streamChan <- event
		}

		if err := sm.sessionStore.UpdateSession(ctx, sess); err != nil {
			return
		}
	}()

	return streamChan, nil
}

// ResumeSession resumes a paused session with an optional rejection reason or tool name.
func (sm *SessionManager) ResumeSession(ctx context.Context, sessionID, confirmation, reason, toolName string) error {
	sm.mux.Lock()
	defer sm.mux.Unlock()

	// Ensure the session runtime exists
	rt, exists := sm.runtimeSessions.Load(sessionID)
	if !exists {
		return errors.New("session not found")
	}

	rt.runtime.Resume(ctx, runtime.ResumeRequest{
		Type:     runtime.ResumeType(confirmation),
		Reason:   reason,
		ToolName: toolName,
	})
	return nil
}

// SteerSession enqueues user messages for mid-turn injection into a running
// session. The messages are picked up by the agent loop after the current tool
// calls finish but before the next LLM call. Returns an error if the session
// is not actively running or if the steer buffer is full.
func (sm *SessionManager) SteerSession(_ context.Context, sessionID string, messages []api.Message) error {
	rt, exists := sm.runtimeSessions.Load(sessionID)
	if !exists {
		return errors.New("session not found or not running")
	}

	localRT := runtime.GetLocalRuntime(rt.runtime)
	if localRT == nil {
		return errors.New("steering not supported for this runtime type")
	}

	for _, msg := range messages {
		if !localRT.Steer(runtime.QueuedMessage{
			Content:      msg.Content,
			MultiContent: msg.MultiContent,
		}) {
			return errors.New("steer queue full")
		}
	}

	return nil
}

// FollowUpSession enqueues user messages for end-of-turn processing in a
// running session. Each message is popped one at a time after the current
// turn finishes, giving each follow-up a full undivided agent turn.
func (sm *SessionManager) FollowUpSession(_ context.Context, sessionID string, messages []api.Message) error {
	rt, exists := sm.runtimeSessions.Load(sessionID)
	if !exists {
		return errors.New("session not found or not running")
	}

	localRT := runtime.GetLocalRuntime(rt.runtime)
	if localRT == nil {
		return errors.New("follow-up not supported for this runtime type")
	}

	for _, msg := range messages {
		if !localRT.FollowUp(runtime.QueuedMessage{
			Content:      msg.Content,
			MultiContent: msg.MultiContent,
		}) {
			return errors.New("follow-up queue full")
		}
	}

	return nil
}

// ResumeElicitation resumes an elicitation request.
func (sm *SessionManager) ResumeElicitation(ctx context.Context, sessionID, action string, content map[string]any) error {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	rt, exists := sm.runtimeSessions.Load(sessionID)
	if !exists {
		return errors.New("session not found")
	}

	return rt.runtime.ResumeElicitation(ctx, tools.ElicitationAction(action), content)
}

// ToggleToolApproval toggles the tool approval mode for a session.
func (sm *SessionManager) ToggleToolApproval(ctx context.Context, sessionID string) error {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	sess, err := sm.sessionStore.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	sess.ToolsApproved = !sess.ToolsApproved

	return sm.sessionStore.UpdateSession(ctx, sess)
}

// UpdateSessionPermissions updates the permissions for a session.
func (sm *SessionManager) UpdateSessionPermissions(ctx context.Context, sessionID string, perms *session.PermissionsConfig) error {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	sess, err := sm.sessionStore.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	sess.Permissions = perms

	return sm.sessionStore.UpdateSession(ctx, sess)
}

// UpdateSessionTitle updates the title for a session.
// If the session is actively running, it also updates the in-memory session
// object to prevent subsequent runtime saves from overwriting the title.
func (sm *SessionManager) UpdateSessionTitle(ctx context.Context, sessionID, title string) error {
	sm.mux.Lock()
	defer sm.mux.Unlock()

	// If session is actively running, update the in-memory session object directly.
	// This ensures the runtime's saveSession won't overwrite our manual edit.
	if rt, ok := sm.runtimeSessions.Load(sessionID); ok && rt.session != nil {
		rt.session.Title = title
		slog.Debug("Updated title for active session", "session_id", sessionID, "title", title)
		return sm.sessionStore.UpdateSession(ctx, rt.session)
	}

	// Session is not actively running, load from store and update
	sess, err := sm.sessionStore.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	sess.Title = title
	return sm.sessionStore.UpdateSession(ctx, sess)
}

// generateTitle generates a title for a session using the sessiontitle package.
// The generated title is stored in the session and persisted to the store.
// A SessionTitleEvent is emitted to notify clients.
func (sm *SessionManager) generateTitle(ctx context.Context, sess *session.Session, gen *sessiontitle.Generator, userMessages []string, events chan<- runtime.Event) {
	if gen == nil || len(userMessages) == 0 {
		return
	}

	title, err := gen.Generate(ctx, sess.ID, userMessages)
	if err != nil {
		slog.Error("Failed to generate session title", "session_id", sess.ID, "error", err)
		return
	}

	if title == "" {
		return
	}

	// Update the in-memory session
	sess.Title = title

	// Persist the title
	if err := sm.sessionStore.UpdateSession(ctx, sess); err != nil {
		slog.Error("Failed to persist generated title", "session_id", sess.ID, "error", err)
		return
	}

	// Emit the title event
	select {
	case events <- runtime.SessionTitle(sess.ID, title):
		slog.Debug("Generated and emitted session title", "session_id", sess.ID, "title", title)
	case <-ctx.Done():
		slog.Debug("Context cancelled while emitting title event", "session_id", sess.ID)
	}
}

func (sm *SessionManager) runtimeForSession(ctx context.Context, sess *session.Session, agentFilename, currentAgent string, rc *config.RuntimeConfig) (runtime.Runtime, *sessiontitle.Generator, error) {
	rt, exists := sm.runtimeSessions.Load(sess.ID)
	if exists && rt.runtime != nil {
		return rt.runtime, rt.titleGen, nil
	}

	t, err := sm.loadTeam(ctx, agentFilename, rc)
	if err != nil {
		return nil, nil, err
	}

	agent, err := t.Agent(currentAgent)
	if err != nil {
		return nil, nil, err
	}
	sess.MaxIterations = agent.MaxIterations()
	sess.MaxConsecutiveToolCalls = agent.MaxConsecutiveToolCalls()
	sess.MaxOldToolCallTokens = agent.MaxOldToolCallTokens()

	opts := []runtime.Opt{
		runtime.WithCurrentAgent(currentAgent),
		runtime.WithManagedOAuth(false),
		runtime.WithSessionStore(sm.sessionStore),
	}
	run, err := runtime.New(t, opts...)
	if err != nil {
		return nil, nil, err
	}

	titleGen := sessiontitle.New(agent.Model(), agent.FallbackModels()...)

	sm.runtimeSessions.Store(sess.ID, &activeRuntimes{
		runtime:  run,
		session:  sess,
		titleGen: titleGen,
	})

	slog.Debug("Runtime created for session", "session_id", sess.ID)

	return run, titleGen, nil
}

func (sm *SessionManager) loadTeam(ctx context.Context, agentFilename string, runConfig *config.RuntimeConfig) (*team.Team, error) {
	agentSource, found := sm.Sources[agentFilename]
	if !found {
		return nil, fmt.Errorf("agent not found: %s", agentFilename)
	}

	return teamloader.Load(ctx, agentSource, runConfig)
}

// GetAgentToolCount loads the agent's team and returns the number of
// tools available to the given agent.
func (sm *SessionManager) GetAgentToolCount(ctx context.Context, agentFilename, agentName string) (int, error) {
	t, err := sm.loadTeam(ctx, agentFilename, sm.runConfig)
	if err != nil {
		return 0, err
	}
	defer func() {
		if stopErr := t.StopToolSets(ctx); stopErr != nil {
			slog.Error("Failed to stop tool sets", "error", stopErr)
		}
	}()

	a, err := t.Agent(agentName)
	if err != nil {
		return 0, err
	}

	agentTools, err := a.Tools(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get tools: %w", err)
	}

	return len(agentTools), nil
}

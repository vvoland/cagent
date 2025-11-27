package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/docker/cagent/pkg/api"
	"github.com/docker/cagent/pkg/concurrent"
	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/tools"
)

type Server struct {
	e              *echo.Echo
	runtimeCancels *concurrent.Map[string, context.CancelFunc]
	sessionStore   session.Store
	runConfig      *config.RuntimeConfig
	sm             *sessionManager
}

func New(sessionStore session.Store, runConfig *config.RuntimeConfig, refreshInterval time.Duration, agentSources config.Sources) (*Server, error) {
	e := echo.New()
	e.Use(middleware.CORS())
	e.Use(middleware.Logger())

	s := &Server{
		e:              e,
		runtimeCancels: concurrent.NewMap[string, context.CancelFunc](),
		sessionStore:   sessionStore,
		runConfig:      runConfig,
		sm:             newSessionManager(agentSources, refreshInterval),
	}

	group := e.Group("/api")

	// List all available agents
	group.GET("/agents", s.getAgents)

	// List all sessions
	group.GET("/sessions", s.getSessions)
	// Get a session by id
	group.GET("/sessions/:id", s.getSession)
	// Resume a session by id
	group.POST("/sessions/:id/resume", s.resumeSession)
	// Create a new session
	group.POST("/sessions", s.createSession)
	// Delete a session
	group.DELETE("/sessions/:id", s.deleteSession)
	// Run an agent loop
	group.POST("/sessions/:id/agent/:agent", s.runAgent)
	group.POST("/sessions/:id/agent/:agent/:agent_name", s.runAgent)
	group.POST("/sessions/:id/elicitation", s.elicitation)

	// Health check endpoint
	group.GET("/ping", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	return s, nil
}

func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	srv := http.Server{
		Handler: s.e,
	}

	if err := srv.Serve(ln); err != nil && ctx.Err() == nil {
		slog.Error("Failed to start server", "error", err)
		return err
	}

	return nil
}

func (s *Server) getAgents(c echo.Context) error {
	agents := []api.Agent{}
	for k, agentSource := range s.sm.sources {
		slog.Debug("API source", "source", agentSource.Name())

		c, err := config.Load(c.Request().Context(), agentSource)
		if err != nil {
			slog.Error("Failed to load config from API source", "key", k, "error", err)
			continue
		}

		var desc string
		if a, ok := c.Agents["root"]; ok {
			desc = a.Description
		} else {
			for _, agent := range c.Agents {
				desc = agent.Description
				break
			}
		}
		switch {
		case len(c.Agents) > 1:
			agents = append(agents, api.Agent{
				Name:        k,
				Multi:       true,
				Description: desc,
			})
		case len(c.Agents) == 1:
			agents = append(agents, api.Agent{
				Name:        k,
				Multi:       false,
				Description: desc,
			})
		default:
			slog.Warn("No agents found in config from API source", "key", k)
			continue
		}
	}

	// Sort agents by name
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].Name < agents[j].Name
	})

	return c.JSON(http.StatusOK, agents)
}

func (s *Server) getSessions(c echo.Context) error {
	sessions, err := s.sessionStore.GetSessions(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to get sessions: %v", err))
	}

	responses := make([]api.SessionsResponse, len(sessions))
	for i, sess := range sessions {
		responses[i] = api.SessionsResponse{
			ID:           sess.ID,
			Title:        sess.Title,
			CreatedAt:    sess.CreatedAt.Format(time.RFC3339),
			NumMessages:  len(sess.GetAllMessages()),
			InputTokens:  sess.InputTokens,
			OutputTokens: sess.OutputTokens,
			WorkingDir:   sess.WorkingDir,
		}
	}
	return c.JSON(http.StatusOK, responses)
}

func (s *Server) createSession(c echo.Context) error {
	var sessionTemplate session.Session
	if err := c.Bind(&sessionTemplate); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
	}

	var opts []session.Opt
	opts = append(opts,
		session.WithMaxIterations(sessionTemplate.MaxIterations),
		session.WithToolsApproved(sessionTemplate.ToolsApproved),
	)

	if wd := strings.TrimSpace(sessionTemplate.WorkingDir); wd != "" {
		absWd, err := filepath.Abs(wd)
		if err != nil {
			slog.Error("Invalid working directory", "error", err)
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid working directory: %v", err))
		}
		info, err := os.Stat(absWd)
		if err != nil {
			slog.Error("Working directory not accessible", "error", err)
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("working directory not accessible: %v", err))
		}
		if !info.IsDir() {
			slog.Error("Working directory is not a directory")
			return echo.NewHTTPError(http.StatusBadRequest, "working directory must be a directory")
		}
		opts = append(opts, session.WithWorkingDir(absWd))
	}

	sess := session.New(opts...)

	if err := s.sessionStore.AddSession(c.Request().Context(), sess); err != nil {
		slog.Error("Failed to persist session", "session_id", sess.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to create session: %v", err))
	}

	return c.JSON(http.StatusOK, sess)
}

func (s *Server) getSession(c echo.Context) error {
	sess, err := s.sessionStore.GetSession(c.Request().Context(), c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("session not found: %v", err))
	}

	params := api.PaginationParams{
		Limit:  api.DefaultLimit,
		Before: c.QueryParam("before"),
	}

	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			params.Limit = limit
		}
	}

	allMessages := sess.GetAllMessages()

	paginatedMessages, pagination, err := api.PaginateMessages(allMessages, params)
	if err != nil {
		slog.Error("Failed to paginate messages", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid pagination parameters: %v", err))
	}

	sr := api.SessionResponse{
		ID:            sess.ID,
		Title:         sess.Title,
		CreatedAt:     sess.CreatedAt,
		Messages:      paginatedMessages,
		ToolsApproved: sess.ToolsApproved,
		InputTokens:   sess.InputTokens,
		OutputTokens:  sess.OutputTokens,
		WorkingDir:    sess.WorkingDir,
		Pagination:    pagination,
	}

	return c.JSON(http.StatusOK, sr)
}

func (s *Server) resumeSession(c echo.Context) error {
	sessionID := c.Param("id")
	var req api.ResumeSessionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
	}

	rt, exists := s.sm.runtimes.Load(sessionID)
	if !exists {
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("runtime not found: %s", sessionID))
	}

	rt.Resume(c.Request().Context(), runtime.ResumeType(req.Confirmation))

	return c.JSON(http.StatusOK, map[string]string{"message": "session resumed"})
}

func (s *Server) deleteSession(c echo.Context) error {
	sessionID := c.Param("id")

	// Cancel the runtime context if it's still running
	if cancel, exists := s.runtimeCancels.Load(sessionID); exists {
		slog.Debug("Cancelling runtime for session", "session_id", sessionID)
		cancel()
		s.runtimeCancels.Delete(sessionID)
	}

	// Clean up the runtime
	if _, exists := s.sm.runtimes.Load(sessionID); exists {
		slog.Debug("Removing runtime for session", "session_id", sessionID)
		s.sm.runtimes.Delete(sessionID)
	}

	// Delete the session from storage
	if err := s.sessionStore.DeleteSession(c.Request().Context(), sessionID); err != nil {
		slog.Error("Failed to delete session", "session_id", sessionID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to delete session: %v", err))
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "session deleted"})
}

func (s *Server) runAgent(c echo.Context) error {
	sessionID := c.Param("id")
	agentFilename := c.Param("agent")
	currentAgent := c.Param("agent_name")
	if currentAgent == "" {
		currentAgent = "root"
	}

	slog.Debug("Running agent", "agent_filename", agentFilename, "session_id", sessionID, "current_agent", currentAgent)

	// Build a per-session team so Filesystem tool can be bound to session working dir
	sess, err := s.sessionStore.GetSession(c.Request().Context(), sessionID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("session not found: %v", err))
	}

	// Copy runConfig and inject per-session working dir override
	rc := s.runConfig.Clone()
	rc.WorkingDir = sess.WorkingDir

	rt, err := s.sm.runtimeForSession(c.Request().Context(), sess, agentFilename, currentAgent, rc)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to get runtime for session: %v", err))
	}

	// Receive messages from the API client
	var messages []api.Message
	if err := json.NewDecoder(c.Request().Body).Decode(&messages); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
	}

	for _, msg := range messages {
		sess.AddMessage(session.UserMessage(msg.Content, msg.MultiContent...))
	}

	if err := s.sessionStore.UpdateSession(c.Request().Context(), sess); err != nil {
		slog.Error("Failed to update session in store", "session_id", sess.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to update session: %v", err))
	}

	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(http.StatusOK)

	streamCtx, cancel := context.WithCancel(c.Request().Context())
	s.runtimeCancels.Store(sess.ID, cancel)
	defer func() {
		s.runtimeCancels.Delete(sess.ID)
	}()

	streamChan := rt.RunStream(streamCtx, sess)
	for event := range streamChan {
		data, err := json.Marshal(event)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to marshal event: %v", err))
		}
		fmt.Fprintf(c.Response(), "data: %s\n\n", string(data))
		c.Response().Flush()
	}

	if err := s.sessionStore.UpdateSession(c.Request().Context(), sess); err != nil {
		slog.Error("Failed to final update session in store", "session_id", sess.ID, "error", err)
	}

	return nil
}

func (s *Server) elicitation(c echo.Context) error {
	sessionID := c.Param("id")
	var req api.ResumeElicitationRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
	}

	rt, exists := s.sm.runtimes.Load(sessionID)
	if !exists {
		return c.JSON(http.StatusNotFound, map[string]string{"error": fmt.Sprintf("runtime not found: %s", sessionID)})
	}

	if err := rt.ResumeElicitation(c.Request().Context(), tools.ElicitationAction(req.Action), req.Content); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to resume elicitation: %v", err))
	}

	return c.JSON(http.StatusOK, nil)
}

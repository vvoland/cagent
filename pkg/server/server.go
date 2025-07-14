package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumpl/cagent/pkg/chat"
	"github.com/rumpl/cagent/pkg/runtime"
	"github.com/rumpl/cagent/pkg/session"
)

type Server struct {
	e            *echo.Echo
	addr         string
	logger       *slog.Logger
	runtimes     map[string]*runtime.Runtime
	sessionStore session.Store
}

type Opt func(*Server)

func WithFrontend(fsys fs.FS) Opt {
	return func(s *Server) {
		assetHandler := http.FileServer(http.FS(fsys))
		s.e.GET("/*", echo.WrapHandler(assetHandler))
	}
}

func New(ctx context.Context, logger *slog.Logger, runtimes map[string]*runtime.Runtime, sessionStore session.Store, listenAddr string, opts ...Opt) (*Server, error) {
	e := echo.New()

	s := &Server{
		e:            e,
		addr:         listenAddr,
		runtimes:     runtimes,
		logger:       logger,
		sessionStore: sessionStore,
	}

	for _, opt := range opts {
		opt(s)
	}

	api := e.Group("/api")
	// List all available agents
	api.GET("/agents", s.agents)
	// List all sessions
	api.GET("/sessions", s.getSessions)
	// Create a new session
	api.POST("/sessions", s.createSession)

	// Run an agent loop
	api.POST("/sessions/:id/agent/:agent", s.runAgent)

	return s, nil
}

func (s *Server) agents(c echo.Context) error {
	agentList := make([]map[string]string, 0)
	for name, agent := range s.runtimes {
		agentList = append(agentList, map[string]string{
			"name":        name,
			"description": agent.CurrentAgent().Description(),
		})
	}
	return c.JSON(http.StatusOK, agentList)
}

func (s *Server) getSessions(c echo.Context) error {
	sessions, err := s.sessionStore.GetSessions(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get sessions"})
	}
	return c.JSON(http.StatusOK, sessions)
}

func (s *Server) createSession(c echo.Context) error {
	sess := session.New(s.logger)

	if err := s.sessionStore.AddSession(c.Request().Context(), sess); err != nil {
		s.logger.Error("Failed to persist session", "session_id", sess.ID, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
	}

	return c.JSON(http.StatusOK, sess)
}

func (s *Server) runAgent(c echo.Context) error {
	agentName := c.Param("agent")

	rt, exists := s.runtimes[agentName]
	if !exists {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "runtime not found"})
	}

	var messages []Message
	if err := json.NewDecoder(c.Request().Body).Decode(&messages); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	// Load session from store
	sess, err := s.sessionStore.GetSession(c.Request().Context(), c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "session not found"})
	}
	sess.SetLogger(s.logger)

	for _, msg := range messages {
		sess.Messages = append(sess.Messages, session.AgentMessage{
			Agent: rt.CurrentAgent(),
			Message: chat.Message{
				Role:    msg.Role,
				Content: msg.Content,
			},
		})
	}

	// Update session in store
	if err := s.sessionStore.UpdateSession(c.Request().Context(), sess); err != nil {
		s.logger.Error("Failed to update session in store", "session_id", sess.ID, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update session"})
	}

	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(http.StatusOK)

	streamChan := rt.RunStream(c.Request().Context(), sess)
	for event := range streamChan {
		data, err := json.Marshal(event)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to marshal event"})
		}
		fmt.Fprintf(c.Response(), "data: %s\n\n", string(data))
		c.Response().Flush()
	}

	// Final update to session store after stream completes
	if err := s.sessionStore.UpdateSession(c.Request().Context(), sess); err != nil {
		s.logger.Error("Failed to final update session in store", "session_id", sess.ID, "error", err)
	}

	return nil
}

func (s *Server) Start() error {
	return s.e.Start(s.addr)
}

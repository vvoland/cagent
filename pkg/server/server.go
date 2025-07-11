package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/rumpl/cagent/pkg/chat"
	"github.com/rumpl/cagent/pkg/runtime"
	"github.com/rumpl/cagent/pkg/session"
)

type Server struct {
	e      *echo.Echo
	addr   string
	logger *slog.Logger

	runtimes map[string]*runtime.Runtime
	sessions map[string]*session.Session

	mux *sync.Mutex
}

type Opt func(*Server)

func WithFrontend(fsys fs.FS) Opt {
	return func(s *Server) {
		assetHandler := http.FileServer(http.FS(fsys))
		s.e.GET("/*", echo.WrapHandler(assetHandler))
	}
}

func New(ctx context.Context, logger *slog.Logger, runtimes map[string]*runtime.Runtime, listenAddr string, opts ...Opt) (*Server, error) {
	e := echo.New()

	s := &Server{e: e, addr: listenAddr, runtimes: runtimes, logger: logger, sessions: make(map[string]*session.Session), mux: &sync.Mutex{}}

	api := e.Group("/api")
	// List all available agents
	api.GET("/agents", s.agents)
	// List all sessions
	api.GET("/sessions", s.getSessions)
	// Create a new session
	api.POST("/sessions", s.createSession)

	// Run an agent loop
	api.POST("/sessions/:id/agent/:agent", s.runAgent)

	for _, opt := range opts {
		opt(s)
	}

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
	s.mux.Lock()
	defer s.mux.Unlock()
	return c.JSON(http.StatusOK, s.sessions)
}

func (s *Server) createSession(c echo.Context) error {
	sess := session.New(s.logger)
	s.mux.Lock()
	s.sessions[sess.ID] = sess
	s.mux.Unlock()
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

	s.mux.Lock()
	sess, ok := s.sessions[c.Param("id")]
	// TODO: We don't really want to lock the whole map, we only want to lock the session.
	defer s.mux.Unlock()
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "session not found"})
	}

	for _, msg := range messages {
		sess.Messages = append(sess.Messages, session.AgentMessage{
			Agent: rt.CurrentAgent(),
			Message: chat.Message{
				Role:    msg.Role,
				Content: msg.Content,
			},
		})
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

	return nil
}

func (s *Server) Start() error {
	return s.e.Start(s.addr)
}

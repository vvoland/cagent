package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/docker/cagent/internal/creator"
	"github.com/docker/cagent/pkg/chat"
	"github.com/docker/cagent/pkg/content"
	"github.com/docker/cagent/pkg/loader"
	"github.com/docker/cagent/pkg/remote"
	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/session"
)

type Server struct {
	e            *echo.Echo
	addr         string
	logger       *slog.Logger
	runtimes     map[string]*runtime.Runtime
	sessionStore session.Store
	agentsDir    string
}

type Opt func(*Server)

func WithFrontend(fsys fs.FS) Opt {
	return func(s *Server) {
		assetHandler := http.FileServer(http.FS(fsys))
		s.e.GET("/*", echo.WrapHandler(assetHandler))
	}
}

func WithAgentsDir(dir string) Opt {
	return func(s *Server) {
		s.agentsDir = dir
	}
}

func New(logger *slog.Logger, runtimes map[string]*runtime.Runtime, sessionStore session.Store, listenAddr string, opts ...Opt) *Server {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.CORS())
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
	api.POST("/agents", s.createAgent)
	api.POST("/agents/pull", s.pullAgent)
	// List all sessions
	api.GET("/sessions", s.getSessions)
	// Create a new session
	api.POST("/sessions", s.createSession)

	api.DELETE("/sessions/:id", s.deleteSession)

	// Run an agent loop
	api.POST("/sessions/:id/agent/:agent", s.runAgent)

	return s
}

type createAgentRequest struct {
	Prompt string `json:"prompt"`
}

func (s *Server) createAgent(c echo.Context) error {
	var req createAgentRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	prompt := req.Prompt

	out, path, err := creator.CreateAgent(c.Request().Context(), s.agentsDir, s.logger, prompt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create agent"})
	}

	team, err := loader.Load(c.Request().Context(), path, s.logger)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load agent"})
	}

	s.runtimes[path], err = runtime.New(s.logger, team, "root")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create runtime"})
	}

	return c.JSON(http.StatusOK, map[string]string{"path": path, "out": out})
}

type pullAgentRequest struct {
	Name string `json:"name"`
}

func (s *Server) pullAgent(c echo.Context) error {
	var req pullAgentRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	_, err := remote.Pull(req.Name)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to pull agent"})
	}

	yaml, err := fromStore(req.Name)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get agent yaml"})
	}

	agentName := strings.ReplaceAll(req.Name, "/", "_")

	fileName := filepath.Join(s.agentsDir, agentName+".yaml")

	if err := os.WriteFile(fileName, []byte(yaml), 0o644); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to write agent yaml to " + fileName + ": " + err.Error()})
	}

	team, err := loader.Load(c.Request().Context(), fileName, s.logger)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load agent"})
	}

	s.runtimes[agentName], err = runtime.New(s.logger, team, "root")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create runtime"})
	}

	return c.JSON(http.StatusOK, map[string]string{"name": agentName})
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

func (s *Server) deleteSession(c echo.Context) error {
	if err := s.sessionStore.DeleteSession(c.Request().Context(), c.Param("id")); err != nil {
		s.logger.Error("Failed to delete session", "session_id", c.Param("id"), "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete session"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "session deleted"})
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

func fromStore(reference string) (string, error) {
	store, err := content.NewStore()
	if err != nil {
		return "", err
	}

	img, err := store.GetArtifactImage(reference)
	if err != nil {
		return "", err
	}

	layers, err := img.Layers()
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	layer := layers[0]
	b, err := layer.Uncompressed()
	if err != nil {
		return "", err
	}

	_, err = io.Copy(&buf, b)
	if err != nil {
		return "", err
	}
	b.Close()

	return buf.String(), nil
}

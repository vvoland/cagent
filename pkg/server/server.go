package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/docker/cagent/internal/creator"
	"github.com/docker/cagent/pkg/chat"
	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/content"
	"github.com/docker/cagent/pkg/loader"
	"github.com/docker/cagent/pkg/remote"
	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/session"
)

type Server struct {
	e            *echo.Echo
	logger       *slog.Logger
	runtimes     map[string]*runtime.Runtime
	sessionStore session.Store
	agentsDir    string
	gateway      string
	envFiles     []string
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

func New(logger *slog.Logger, runtimes map[string]*runtime.Runtime, sessionStore session.Store, envFiles []string, gateway string, opts ...Opt) *Server {
	e := echo.New()
	e.Use(middleware.CORS())
	s := &Server{
		e:            e,
		runtimes:     runtimes,
		logger:       logger,
		sessionStore: sessionStore,
		gateway:      gateway,
		envFiles:     envFiles,
	}

	for _, opt := range opts {
		opt(s)
	}

	api := e.Group("/api")

	// List all available agents
	api.GET("/agents", s.agents)
	// Get an agent by id
	api.GET("/agents/:id", s.getAgent)
	// Create a new agent
	api.POST("/agents", s.createAgent)
	// Pull an agent from a remote registry
	api.POST("/agents/pull", s.pullAgent)
	// List all sessions
	api.GET("/sessions", s.getSessions)
	// Get a session by id
	api.GET("/sessions/:id", s.getSession)
	// Create a new session and run an agent loop
	api.POST("/sessions", s.createSession)
	// Delete a session
	api.DELETE("/sessions/:id", s.deleteSession)

	// Run an agent loop
	api.POST("/sessions/:id/agent/:agent", s.runAgent)

	return s
}

func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	srv := http.Server{
		Handler: s.e,
	}

	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		s.logger.Error("Failed to start server", "error", err)
		return err
	}

	return nil
}

type createAgentRequest struct {
	Prompt string `json:"prompt"`
}

func (s *Server) getAgent(c echo.Context) error {
	path := filepath.Join(s.agentsDir, c.Param("id"))
	if !strings.HasSuffix(path, ".yaml") {
		path += ".yaml"
	}
	cfg, err := config.LoadConfig(path)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "agent not found"})
	}

	return c.JSON(http.StatusOK, cfg)
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

	team, err := loader.Load(c.Request().Context(), path, s.envFiles, s.gateway, s.logger)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load agent"})
	}

	if err := team.StartToolSets(context.TODO()); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to start tool sets"})
	}

	agentName := filepath.Base(path)
	s.runtimes[agentName], err = runtime.New(s.logger, team, "root")
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

	team, err := loader.Load(c.Request().Context(), fileName, s.envFiles, s.gateway, s.logger)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load agent"})
	}

	if err := team.StartToolSets(c.Request().Context()); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to start tool sets"})
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

type sessionResponse struct {
	ID          string `json:"id"`
	CreatedAt   string `json:"created_at"`
	NumMessages int    `json:"num_messages"`
}

func (s *Server) getSessions(c echo.Context) error {
	sessions, err := s.sessionStore.GetSessions(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get sessions"})
	}

	responses := make([]sessionResponse, len(sessions))
	for i, sess := range sessions {
		responses[i] = sessionResponse{
			ID:          sess.ID,
			CreatedAt:   sess.CreatedAt.Format(time.RFC3339),
			NumMessages: len(sess.Messages),
		}
	}
	return c.JSON(http.StatusOK, responses)
}

func (s *Server) createSession(c echo.Context) error {
	sess := session.New(s.logger)

	if err := s.sessionStore.AddSession(c.Request().Context(), sess); err != nil {
		s.logger.Error("Failed to persist session", "session_id", sess.ID, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
	}

	return c.JSON(http.StatusOK, sess)
}

func (s *Server) getSession(c echo.Context) error {
	sess, err := s.sessionStore.GetSession(c.Request().Context(), c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "session not found"})
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

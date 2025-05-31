package root

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rumpl/cagent/pkg/agent"
	"github.com/rumpl/cagent/pkg/chat"
	"github.com/rumpl/cagent/pkg/config"
	"github.com/rumpl/cagent/pkg/runtime"
	"github.com/rumpl/cagent/pkg/session"
	"github.com/spf13/cobra"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

var (
	listenAddr    string
	agentsDir     string
	runtimes      map[string]*runtime.Runtime
	runtimeAgents map[string]map[string]*agent.Agent
)

// NewWebCmd creates a new web command
func NewWebCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "web",
		Short: "Start a web server",
		Long:  `Start a web server that exposes the agent via an HTTP API`,
		RunE:  runWebCommand,
	}

	cmd.PersistentFlags().StringVarP(&configFile, "config", "c", "agent.yaml", "Path to the configuration file")
	cmd.PersistentFlags().StringVarP(&agentsDir, "agents-dir", "d", "", "Directory containing agent configurations")
	cmd.PersistentFlags().StringVarP(&listenAddr, "listen", "l", ":8080", "Address to listen on")

	return cmd
}

func runWebCommand(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := slog.Default()
	logger.Debug("Starting web server", "agents-dir", agentsDir)

	var cfg *config.Config
	var err error

	if agentsDir != "" {
		runtimes = make(map[string]*runtime.Runtime)
		runtimeAgents = make(map[string]map[string]*agent.Agent)

		entries, err := os.ReadDir(agentsDir)
		if err != nil {
			return fmt.Errorf("failed to read directory: %w", err)
		}

		for _, entry := range entries {
			agents := make(map[string]*agent.Agent)
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
				configPath := filepath.Join(agentsDir, entry.Name())
				cfg, err = config.LoadConfig(configPath)
				if err != nil {
					logger.Warn("Failed to load config", "file", entry.Name(), "error", err)
					continue
				}

				fileAgents, err := config.Agents(configPath)
				if err != nil {
					logger.Warn("Failed to load agents", "file", entry.Name(), "error", err)
					continue
				}

				// Create runtimes for each agent in this file
				for name := range fileAgents {
					if _, exists := agents[name]; exists {
						return fmt.Errorf("duplicate agent name '%s' found in %s", name, configPath)
					}
					agents[name] = fileAgents[name]

					runtimeAgents[entry.Name()] = fileAgents

					// Create a runtime with only the agents from this file
					fileAgentsMap := make(map[string]*agent.Agent)
					for n, a := range fileAgents {
						fileAgentsMap[n] = a
					}

					rt, err := runtime.New(cfg, logger, fileAgentsMap, name)
					if err != nil {
						return fmt.Errorf("failed to create runtime for agent %s from file %s: %w", name, entry.Name(), err)
					}
					runtimes[entry.Name()] = rt
				}
			}
		}

		// if len(agents) == 0 {
		// 	return fmt.Errorf("no valid agent configurations found in directory %s", agentsDir)
		// }
	} else {
		cfg, err = config.LoadConfig(configFile)
		if err != nil {
			return err
		}

		agents, err := config.Agents(configFile)
		if err != nil {
			return err
		}

		// Initialize runtimes for single config file
		runtimes = make(map[string]*runtime.Runtime)
		for name := range agents {
			rt, err := runtime.New(cfg, logger, agents, name)
			if err != nil {
				return fmt.Errorf("failed to create runtime for agent %s: %w", name, err)
			}
			runtimes[name] = rt
		}
	}

	e := echo.New()
	sessions := make(map[string]*session.Session)

	// List all available agents
	e.GET("/agents", func(c echo.Context) error {
		agentList := make([]map[string]string, 0)
		for name, agent := range runtimes {
			agentList = append(agentList, map[string]string{
				"name":        name,
				"description": agent.CurrentAgent().Description(),
			})
		}
		return c.JSON(http.StatusOK, agentList)
	})

	e.GET("/sessions", func(c echo.Context) error {
		return c.JSON(http.StatusOK, sessions)
	})

	e.POST("/sessions/:agent", func(c echo.Context) error {
		agentName := c.Param("agent")
		agents, exists := runtimeAgents[agentName]
		if !exists {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "runtime not found"})
		}

		sess := session.New(agents)
		sessions[sess.ID] = sess
		return c.JSON(http.StatusOK, sess)
	})

	e.POST("/sessions/:id/agent/:agent", func(c echo.Context) error {
		agentName := c.Param("agent")

		rt, exists := runtimes[agentName]
		if !exists {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "runtime not found"})
		}

		var messages []Message
		if err := json.NewDecoder(c.Request().Body).Decode(&messages); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		}

		sess, ok := sessions[c.Param("id")]
		if !ok {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "session not found"})
		}

		for _, msg := range messages {
			sess.Messages = append(sess.Messages, session.AgentMessage{
				Agent: rt.CurrentAgent(),
				Message: chat.ChatCompletionMessage{
					Role:    msg.Role,
					Content: msg.Content,
				},
			})
		}

		c.Response().Header().Set("Content-Type", "text/event-stream")
		c.Response().Header().Set("Cache-Control", "no-cache")
		c.Response().Header().Set("Connection", "keep-alive")
		c.Response().WriteHeader(http.StatusOK)

		streamChan := rt.RunStream(ctx, sess)
		for event := range streamChan {
			data, _ := json.Marshal(event)
			fmt.Fprintf(c.Response(), "data: %s\n\n", string(data))
			c.Response().Flush()
		}

		return nil
	})

	return e.Start(listenAddr)
}

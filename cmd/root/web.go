package root

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
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

// NewWebCmd creates a new web command
func NewWebCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "web",
		Short: "Start a web server",
		Long:  `Start a web server that exposes the agent via an HTTP API`,
		RunE:  runWebCommand,
	}

	cmd.PersistentFlags().StringVarP(&configFile, "config", "c", "agent.yaml", "Path to the configuration file")
	cmd.PersistentFlags().StringVarP(&agentName, "agent", "a", "root", "Name of the agent to run")
	cmd.PersistentFlags().StringVarP(&listenAddr, "listen", "l", ":8080", "Address to listen on")

	return cmd
}

var listenAddr string

func runWebCommand(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := slog.Default()
	logger.Debug("Starting web server", "agent", agentName)

	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return err
	}

	agents, err := config.Agents(configFile)
	if err != nil {
		return err
	}
	e := echo.New()
	sessions := make(map[string]*session.Session)
	e.GET("/sessions", func(c echo.Context) error {
		return c.JSON(http.StatusOK, sessions)
	})

	e.POST("/sessions", func(c echo.Context) error {
		sess := session.New(agents)
		sessions[sess.ID] = sess
		return c.JSON(http.StatusOK, sess)
	})

	e.POST("/sessions/:id/agent", func(c echo.Context) error {
		rt, err := runtime.New(cfg, logger, agents, agentName)
		if err != nil {
			return err
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
				Agent: agents[agentName],
				Message: chat.ChatCompletionMessage{
					Role:    msg.Role,
					Content: msg.Content,
				},
			})
		}

		fmt.Println(sess.Messages)

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

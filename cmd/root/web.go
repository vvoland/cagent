package root

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/docker/cagent/pkg/chat"
	"github.com/docker/cagent/pkg/loader"
	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/server"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/web"
)

type Message struct {
	Role    chat.MessageRole `json:"role"`
	Content string           `json:"content"`
}

var (
	listenAddr string
	sessionDb  string
	envFiles   []string
)

// NewWebCmd creates a new web command
func NewWebCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "web <agent-file>|<agents-dir>",
		Short:   "Start a web server",
		Long:    `Start a web server that exposes the agents via an HTTP API`,
		Example: `cagent web /path/to/agents --listen :8080`,
		Args:    cobra.ExactArgs(1),
		RunE:    runWebCommand,
	}

	cmd.PersistentFlags().StringVarP(&listenAddr, "listen", "l", ":8080", "Address to listen on")
	cmd.PersistentFlags().StringVarP(&sessionDb, "session-db", "s", "session.db", "Path to the session database")
	cmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug logging")
	cmd.PersistentFlags().StringSliceVar(&envFiles, "env-from-file", nil, "Set environment variables from file")

	return cmd
}

func runWebCommand(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	agentsPath := args[0]

	logLevel := slog.LevelInfo
	if debugMode {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))

	agents, err := findAgents(agentsPath)
	if err != nil {
		return fmt.Errorf("failed to find agents: %w", err)
	}

	logger.Debug("Starting API server", "agents", agentsPath, "debug_mode", debugMode)

	runtimes := make(map[string]*runtime.Runtime)

	for _, agentPath := range agents {
		fileTeam, err := loader.Load(ctx, agentPath, envFiles, logger)
		if err != nil {
			logger.Warn("Failed to load agent", "file", agentPath, "error", err)
			continue
		}

		if err := fileTeam.StartToolSets(ctx); err != nil {
			return fmt.Errorf("failed to start tool sets: %w", err)
		}

		filename := filepath.Base(agentPath)
		rt, err := runtime.New(logger, fileTeam, "root")
		if err != nil {
			return fmt.Errorf("failed to create runtime for file %s: %w", filename, err)
		}
		runtimes[filename] = rt
	}

	defer func() {
		for _, rt := range runtimes {
			if err := rt.Team().StopToolSets(); err != nil {
				logger.Error("Failed to stop tool sets", "error", err)
			}
		}
	}()

	fsys, err := fs.Sub(web.WebContent, "dist")
	if err != nil {
		return err
	}

	sessionStore, err := session.NewSQLiteSessionStore(sessionDb)
	if err != nil {
		return fmt.Errorf("failed to create session store: %w", err)
	}

	s := server.New(logger, runtimes, sessionStore, server.WithFrontend(fsys))
	return s.ListenAndServe(ctx, listenAddr)
}

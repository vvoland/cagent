package root

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/docker/cagent/pkg/loader"
	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/server"
	"github.com/docker/cagent/pkg/session"
)

// NewWebCmd creates a new web command
func NewApiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api <agent-file>|<agents-dir>",
		Short: "Start the API server",
		Long:  `Start the API server that exposes the agent via an HTTP API`,
		Args:  cobra.ExactArgs(1),
		RunE:  runApiCommand,
	}

	cmd.PersistentFlags().StringVarP(&listenAddr, "listen", "l", ":8080", "Address to listen on")
	cmd.PersistentFlags().StringVarP(&sessionDb, "session-db", "s", "session.db", "Path to the session database")
	cmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug logging")
	cmd.PersistentFlags().StringSliceVar(&envFiles, "env-from-file", nil, "Set environment variables from file")
	cmd.PersistentFlags().StringVar(&gateway, "gateway", "", "Set the gateway address")

	return cmd
}

func runApiCommand(cmd *cobra.Command, args []string) error {
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
		fileTeam, err := loader.Load(ctx, agentPath, envFiles, gateway, logger)
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

	sessionStore, err := session.NewSQLiteSessionStore(sessionDb)
	if err != nil {
		return fmt.Errorf("failed to create session store: %w", err)
	}

	var opts []server.Opt
	stat, err := os.Stat(agentsPath)
	if err != nil {
		return fmt.Errorf("failed to stat agents path: %w", err)
	}
	if stat.IsDir() {
		opts = append(opts, server.WithAgentsDir(agentsPath))
	}

	s := server.New(logger, runtimes, sessionStore, gateway, opts...)
	return s.ListenAndServe(ctx, listenAddr)
}

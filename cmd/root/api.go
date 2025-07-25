package root

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

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

	return cmd
}

func runApiCommand(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Configure logger based on debug flag
	logLevel := slog.LevelInfo
	if debugMode {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))

	// Create session store
	sessionStore, err := session.NewSQLiteSessionStore(sessionDb)
	if err != nil {
		return fmt.Errorf("failed to create session store: %w", err)
	}

	agentsPath := args[0]
	stat, err := os.Stat(agentsPath)
	if err != nil {
		return fmt.Errorf("failed to stat agents path: %w", err)
	}

	if stat.IsDir() {
		agentsDir := agentsPath
		logger.Debug("Starting API server", "agents-dir", agentsDir, "debug_mode", debugMode)

		runtimes = make(map[string]*runtime.Runtime)

		entries, err := os.ReadDir(agentsDir)
		if err != nil {
			return fmt.Errorf("failed to read directory: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
				continue
			}
			configPath := filepath.Join(agentsDir, entry.Name())
			fileTeam, err := loader.Load(ctx, configPath, envFiles, logger)
			if err != nil {
				logger.Warn("Failed to load agents", "file", entry.Name(), "error", err)
				continue
			}

			if err := fileTeam.StartToolSets(ctx); err != nil {
				return fmt.Errorf("failed to start tool sets: %w", err)
			}

			rt, err := runtime.New(logger, fileTeam, "root")
			if err != nil {
				return fmt.Errorf("failed to create runtime for file %s: %w", entry.Name(), err)
			}
			runtimes[entry.Name()] = rt
		}

		defer func() {
			for _, rt := range runtimes {
				if err := rt.Team().StopToolSets(); err != nil {
					logger.Error("Failed to stop tool sets", "error", err)
				}
			}
		}()

		s := server.New(logger, runtimes, sessionStore, server.WithAgentsDir(agentsDir))
		return s.ListenAndServe(ctx, listenAddr)
	}

	logger.Debug("Starting API server", "agent-file", agentsPath, "debug_mode", debugMode)

	t, err := loader.Load(ctx, agentsPath, envFiles, logger)
	if err != nil {
		return err
	}
	defer func() {
		for _, rt := range runtimes {
			if err := rt.Team().StopToolSets(); err != nil {
				logger.Error("Failed to stop tool sets", "error", err)
			}
		}
	}()

	if err := t.StartToolSets(ctx); err != nil {
		return fmt.Errorf("failed to start tool sets: %w", err)
	}

	// Initialize runtimes for single config file
	runtimes = make(map[string]*runtime.Runtime)
	rt, err := runtime.New(logger, t, "root")
	if err != nil {
		return err
	}
	runtimes[filepath.Base(agentsPath)] = rt

	s := server.New(logger, runtimes, sessionStore)
	return s.ListenAndServe(ctx, listenAddr)
}

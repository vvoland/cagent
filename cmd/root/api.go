package root

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"maps"

	"github.com/docker/cagent/pkg/agent"
	"github.com/docker/cagent/pkg/loader"
	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/server"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/team"
	"github.com/spf13/cobra"
)

// NewWebCmd creates a new web command
func NewApiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api <agent-name>",
		Short: "Start the API server",
		Long:  `Start the API server that exposes the agent via an HTTP API`,
		Args:  cobra.ExactArgs(1),
		RunE:  runApiCommand,
	}

	cmd.PersistentFlags().StringVarP(&agentsDir, "agents-dir", "d", "", "Directory containing agent configurations")
	cmd.PersistentFlags().StringVarP(&listenAddr, "listen", "l", ":8080", "Address to listen on")
	cmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug logging")

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

	logger.Debug("Starting API server", "agents-dir", agentsDir, "debug_mode", debugMode)

	// Create session store
	sessionStore, err := session.NewSQLiteSessionStore("sessions.db")
	if err != nil {
		return fmt.Errorf("failed to create session store: %w", err)
	}

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
				fileTeam, err := loader.Load(ctx, configPath, logger)
				if err != nil {
					logger.Warn("Failed to load agents", "file", entry.Name(), "error", err)
					continue
				}

				// Create runtimes for each agent in this file
				for name := range fileTeam.Agents() {
					if _, exists := agents[name]; exists {
						return fmt.Errorf("duplicate agent name '%s' found in %s", name, configPath)
					}
					agents[name] = fileTeam.Get(name)

					runtimeAgents[entry.Name()] = fileTeam.Agents()

					// Create a runtime with only the agents from this file
					fileAgentsMap := make(map[string]*agent.Agent, fileTeam.Size())
					maps.Copy(fileAgentsMap, fileTeam.Agents())

					rt, err := runtime.New(logger, team.New(fileAgentsMap), "root")
					if err != nil {
						return fmt.Errorf("failed to create runtime for agent %s from file %s: %w", name, entry.Name(), err)
					}
					runtimes[entry.Name()] = rt
				}
			}
		}
	} else {
		t, err := loader.Load(ctx, args[0], logger)
		if err != nil {
			return err
		}

		// Initialize runtimes for single config file
		runtimes = make(map[string]*runtime.Runtime)
		rt, err := runtime.New(logger, t, "root")
		if err != nil {
			return err
		}
		runtimes[filepath.Base(args[0])] = rt
	}

	s, err := server.New(ctx, logger, runtimes, sessionStore, listenAddr)
	if err != nil {
		return err
	}

	return s.Start()
}

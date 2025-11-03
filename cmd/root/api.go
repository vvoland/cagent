package root

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/server"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/teamloader"
	"github.com/docker/cagent/pkg/telemetry"
)

type apiFlags struct {
	listenAddr string
	sessionDB  string
	runConfig  config.RuntimeConfig
}

func newAPICmd() *cobra.Command {
	var flags apiFlags

	cmd := &cobra.Command{
		Use:   "api <agent-file>|<agents-dir>",
		Short: "Start the API server",
		Long:  `Start the API server that exposes the agent via an HTTP API`,
		Args:  cobra.ExactArgs(1),
		RunE:  flags.runAPICommand,
	}

	cmd.PersistentFlags().StringVarP(&flags.listenAddr, "listen", "l", ":8080", "Address to listen on")
	cmd.PersistentFlags().StringVarP(&flags.sessionDB, "session-db", "s", "session.db", "Path to the session database")

	addRuntimeConfigFlags(cmd, &flags.runConfig)

	return cmd
}

func (f *apiFlags) runAPICommand(cmd *cobra.Command, args []string) error {
	telemetry.TrackCommand("api", args)

	ctx := cmd.Context()
	agentsPath := args[0]

	// Make sure no question is ever asked to the user in api mode.
	os.Stdin = nil

	ln, err := server.Listen(ctx, f.listenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", f.listenAddr, err)
	}
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	if _, ok := ln.(*net.TCPListener); ok {
		slog.Info("Listening on http://localhost" + f.listenAddr)
	} else {
		slog.Info("Listening on " + f.listenAddr)
	}

	slog.Debug("Starting server", "agents", agentsPath)

	sessionStore, err := session.NewSQLiteSessionStore(f.sessionDB)
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
	} else {
		opts = append(opts, server.WithAgentsDir(filepath.Dir(agentsPath)))
	}

	teams, err := teamloader.LoadTeams(ctx, agentsPath, f.runConfig)
	if err != nil {
		return fmt.Errorf("failed to load teams: %w", err)
	}
	defer func() {
		for _, team := range teams {
			if err := team.StopToolSets(ctx); err != nil {
				slog.Error("Failed to stop tool sets", "error", err)
			}
		}
	}()

	s, err := server.New(sessionStore, f.runConfig, teams, opts...)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	return s.Serve(ctx, ln)
}

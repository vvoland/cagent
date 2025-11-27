package root

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/docker/cagent/pkg/cli"
	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/server"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/telemetry"
)

type apiFlags struct {
	listenAddr       string
	sessionDB        string
	pullIntervalMins int
	runConfig        config.RuntimeConfig
}

func newAPICmd() *cobra.Command {
	var flags apiFlags

	cmd := &cobra.Command{
		Use:     "api <agent-file>|<agents-dir>",
		Short:   "Start the cagent API server",
		Long:    `Start the API server that exposes the agent via a cagent-specific HTTP API`,
		GroupID: "server",
		Args:    cobra.ExactArgs(1),
		RunE:    flags.runAPICommand,
	}

	cmd.PersistentFlags().StringVarP(&flags.listenAddr, "listen", "l", ":8080", "Address to listen on")
	cmd.PersistentFlags().StringVarP(&flags.sessionDB, "session-db", "s", "session.db", "Path to the session database")
	cmd.PersistentFlags().IntVar(&flags.pullIntervalMins, "pull-interval", 0, "Auto-pull OCI reference every N minutes (0 = disabled)")
	addRuntimeConfigFlags(cmd, &flags.runConfig)

	return cmd
}

func (f *apiFlags) runAPICommand(cmd *cobra.Command, args []string) error {
	telemetry.TrackCommand("api", args)

	ctx := cmd.Context()
	out := cli.NewPrinter(cmd.OutOrStdout())
	agentsPath := args[0]

	// Make sure no question is ever asked to the user in api mode.
	os.Stdin = nil

	if f.pullIntervalMins > 0 && !config.IsOCIReference(agentsPath) {
		return fmt.Errorf("--pull-interval flag can only be used with OCI references, not local files")
	}

	ln, err := server.Listen(ctx, f.listenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", f.listenAddr, err)
	}
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	out.Println("Listening on " + ln.Addr().String())

	slog.Debug("Starting server", "agents", agentsPath, "addr", ln.Addr().String())

	sessionStore, err := session.NewSQLiteSessionStore(f.sessionDB)
	if err != nil {
		return fmt.Errorf("failed to create session store: %w", err)
	}

	sources, err := config.ResolveSources(agentsPath)
	if err != nil {
		return fmt.Errorf("failed to resolve agent sources: %w", err)
	}
	s, err := server.New(sessionStore, &f.runConfig, sources)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// TODO(rumpl): implement pull interval

	return s.Serve(ctx, ln)
}

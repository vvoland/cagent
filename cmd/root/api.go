package root

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/docker/cagent/pkg/cli"
	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/connectrpc"
	"github.com/docker/cagent/pkg/server"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/telemetry"
)

// shouldMonitorStdin determines if we should monitor stdin for parent process death.
// This is only meaningful when:
// 1. We're not running as PID 1 or direct child of init (ppid > 1)
// 2. stdin is a pipe (indicating we were spawned by a parent with piped stdio)
//
// In containers, stdin is typically /dev/null or closed, so we skip monitoring
// to avoid immediate shutdown.
func shouldMonitorStdin(ppid int, stdin *os.File) bool {
	// Skip if running as PID 1 or direct child of init (common in containers/systemd)
	if ppid <= 1 {
		slog.Debug("Skipping stdin monitor: running as init or direct child of init", "ppid", ppid)
		return false
	}

	if stdin == nil {
		return false
	}

	// Check if stdin is a pipe
	fi, err := stdin.Stat()
	if err != nil {
		slog.Debug("Skipping stdin monitor: cannot stat stdin", "error", err)
		return false
	}

	// Only monitor if stdin is a pipe (parent process has piped stdio to us)
	if fi.Mode()&os.ModeNamedPipe == 0 {
		slog.Debug("Skipping stdin monitor: stdin is not a pipe", "mode", fi.Mode())
		return false
	}

	slog.Debug("Enabling stdin monitor: stdin is a pipe from parent process", "ppid", ppid)
	return true
}

type apiFlags struct {
	listenAddr       string
	sessionDB        string
	pullIntervalMins int
	fakeResponses    string
	recordPath       string
	connectRPC       bool
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
	cmd.PersistentFlags().StringVar(&flags.fakeResponses, "fake", "", "Replay AI responses from cassette file (for testing)")
	cmd.PersistentFlags().StringVar(&flags.recordPath, "record", "", "Record AI API interactions to cassette file")
	cmd.PersistentFlags().BoolVar(&flags.connectRPC, "connect-rpc", false, "Use Connect-RPC protocol instead of HTTP/JSON API")
	cmd.MarkFlagsMutuallyExclusive("fake", "record")
	addRuntimeConfigFlags(cmd, &flags.runConfig)

	return cmd
}

// monitorStdin monitors stdin for EOF, which indicates the parent process has died.
// When spawned with piped stdio, stdin closes when the parent process dies.
func monitorStdin(ctx context.Context, cancel context.CancelFunc, stdin *os.File) {
	// Close stdin when context is cancelled to unblock the read
	go func() {
		<-ctx.Done()
		stdin.Close()
	}()

	buf := make([]byte, 1)
	for {
		n, err := stdin.Read(buf)
		if err != nil || n == 0 {
			// Only log and cancel if context isn't already done (parent died)
			if ctx.Err() == nil {
				slog.Info("stdin closed, parent process likely died, shutting down")
				cancel()
			}
			return
		}
	}
}

func (f *apiFlags) runAPICommand(cmd *cobra.Command, args []string) error {
	f.runConfig.ModelsGateway = "http://localhost:7777"

	telemetry.TrackCommand("api", args)

	ctx := cmd.Context()

	// Save stdin before redirecting it, so we can optionally monitor for parent death
	stdin := os.Stdin

	out := cli.NewPrinter(cmd.OutOrStdout())
	agentsPath := args[0]

	// Redirect stdin to /dev/null to prevent interactive prompts in API mode.
	// We use /dev/null instead of nil to avoid panics in code that calls os.Stdin.Fd().
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		slog.Warn("Failed to open /dev/null, setting stdin to nil", "error", err)
	} else {
		os.Stdin = devNull
		defer devNull.Close()
	}

	// Monitor stdin for EOF to detect parent process death.
	// Only enabled when stdin is a pipe (indicating we were spawned by a parent process).
	// In containers, stdin is typically /dev/null or closed, so we skip monitoring.
	if shouldMonitorStdin(os.Getppid(), stdin) {
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		defer cancel()
		go monitorStdin(ctx, cancel, stdin)
	}

	// Start fake proxy if --fake is specified
	cleanup, err := setupFakeProxy(f.fakeResponses, 0, &f.runConfig)
	if err != nil {
		return err
	}
	defer func() {
		if err := cleanup(); err != nil {
			slog.Error("Failed to cleanup fake proxy", "error", err)
		}
	}()

	// Start recording proxy if --record is specified
	if _, cleanup, err := setupRecordingProxy(f.recordPath, &f.runConfig); err != nil {
		return err
	} else if cleanup != nil {
		defer cleanup()
	}

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
		return fmt.Errorf("creating session store: %w", err)
	}

	sources, err := config.ResolveSources(agentsPath)
	if err != nil {
		return fmt.Errorf("resolving agent sources: %w", err)
	}

	if f.connectRPC {
		s, err := connectrpc.New(ctx, sessionStore, &f.runConfig, time.Duration(f.pullIntervalMins)*time.Minute, sources)
		if err != nil {
			return fmt.Errorf("creating Connect-RPC server: %w", err)
		}
		return s.Serve(ctx, ln)
	}

	s, err := server.New(ctx, sessionStore, &f.runConfig, time.Duration(f.pullIntervalMins)*time.Minute, sources)
	if err != nil {
		return fmt.Errorf("creating server: %w", err)
	}

	return s.Serve(ctx, ln)
}

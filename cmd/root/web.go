package root

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

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
	agentsDir  string
	runtimes   map[string]*runtime.Runtime
)

// NewWebCmd creates a new web command
func NewWebCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "web session.db",
		Short:   "Start a web server",
		Long:    `Start a web server that exposes the agents via an HTTP API`,
		Example: `cagent web /tmp/session.db --agents-dir /path/to/agents --listen :8080`,
		Args:    cobra.ExactArgs(1),
		RunE:    runWebCommand,
	}

	cmd.PersistentFlags().StringVarP(&agentsDir, "agents-dir", "d", "", "Directory containing agent configurations")
	cmd.PersistentFlags().StringVarP(&listenAddr, "listen", "l", ":8080", "Address to listen on")
	cmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug logging")

	return cmd
}

func runWebCommand(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	logLevel := slog.LevelInfo
	if debugMode {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))

	logger.Debug("Starting web server", "agents-dir", agentsDir, "debug_mode", debugMode)

	// Create session store
	sessionStore, err := session.NewSQLiteSessionStore(args[0])
	if err != nil {
		return fmt.Errorf("failed to create session store: %w", err)
	}

	if agentsDir != "" {
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
			fileTeam, err := loader.Load(ctx, configPath, logger)
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

	if len(runtimes) == 0 {
		return fmt.Errorf("no agents found")
	}

	fsys, err := fs.Sub(web.WebContent, "dist")
	if err != nil {
		return err
	}

	s := server.New(logger, runtimes, sessionStore, listenAddr, server.WithFrontend(fsys))
	return s.Start()
}

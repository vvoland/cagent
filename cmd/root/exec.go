package root

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"

	"github.com/docker/cagent/pkg/remote"
	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/teamloader"
)

func NewExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec <agent-name>",
		Short: "Execute an agent",
		Args:  cobra.ExactArgs(1),
		RunE:  execCommand,
	}

	cmd.PersistentFlags().StringVarP(&agentName, "agent", "a", "root", "Name of the agent to run")
	cmd.PersistentFlags().StringSliceVar(&runConfig.EnvFiles, "env-from-file", nil, "Set environment variables from file")
	cmd.PersistentFlags().StringVar(&workingDir, "working-dir", "", "Set the working directory for the session (applies to tools and relative paths)")
	cmd.PersistentFlags().BoolVar(&autoApprove, "yolo", false, "Automatically approve all tool calls without prompting")
	cmd.PersistentFlags().StringVar(&attachmentPath, "attach", "", "Attach an image file to the message")
	addGatewayFlags(cmd)

	return cmd
}

func execCommand(_ *cobra.Command, args []string) error {
	ctx := context.Background()

	slog.Debug("Starting agent", "agent", agentName, "debug_mode", debugMode)

	agentFilename := args[0]
	if !strings.Contains(agentFilename, "\n") {
		if abs, err := filepath.Abs(agentFilename); err == nil {
			agentFilename = abs
		}
	}

	if enableOtel {
		shutdown, err := initOTelSDK(ctx)
		if err != nil {
			slog.Warn("Failed to initialize OpenTelemetry SDK", "error", err)
		} else if shutdown != nil {
			defer func() {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := shutdown(shutdownCtx); err != nil {
					slog.Warn("Failed to shutdown OpenTelemetry SDK", "error", err)
				}
			}()
			slog.Debug("OpenTelemetry SDK initialized successfully")
		}
	}

	// If working-dir was provided, validate and change process working directory
	if workingDir != "" {
		absWd, err := filepath.Abs(workingDir)
		if err != nil {
			return fmt.Errorf("invalid working directory: %w", err)
		}
		info, err := os.Stat(absWd)
		if err != nil || !info.IsDir() {
			return fmt.Errorf("working directory does not exist or is not a directory: %s", absWd)
		}
		if err := os.Chdir(absWd); err != nil {
			return fmt.Errorf("failed to change working directory: %w", err)
		}
		_ = os.Setenv("PWD", absWd)
		slog.Debug("Working directory set", "dir", absWd)
	}

	// Determine how to obtain the agent definition
	ext := strings.ToLower(filepath.Ext(agentFilename))
	if ext == ".yaml" || ext == ".yml" {
		// Treat as local YAML file: resolve to absolute path so later chdir doesn't break it
		if !strings.Contains(agentFilename, "\n") {
			if abs, err := filepath.Abs(agentFilename); err == nil {
				agentFilename = abs
			}
		}
		if !fileExists(agentFilename) {
			return fmt.Errorf("agent file not found: %s", agentFilename)
		}
	} else {
		// Treat as an OCI image reference. Try local store first, otherwise pull then load.
		a, err := fromStore(agentFilename)
		if err != nil {
			fmt.Println("Pulling agent ", agentFilename)
			if _, pullErr := remote.Pull(agentFilename); pullErr != nil {
				return fmt.Errorf("failed to pull OCI image %s: %w", agentFilename, pullErr)
			}
			// Retry after pull
			a, err = fromStore(agentFilename)
			if err != nil {
				return fmt.Errorf("failed to load agent from store after pull: %w", err)
			}
		}

		// Write the fetched content to a temporary YAML file
		tmpFile, err := os.CreateTemp("", "agentfile-*.yaml")
		if err != nil {
			return err
		}
		defer os.Remove(tmpFile.Name())
		if _, err := tmpFile.WriteString(a); err != nil {
			tmpFile.Close()
			return err
		}
		if err := tmpFile.Close(); err != nil {
			return err
		}
		agentFilename = tmpFile.Name()
	}

	agents, err := teamloader.Load(ctx, agentFilename, runConfig)
	if err != nil {
		return err
	}
	defer func() {
		if err := agents.StopToolSets(); err != nil {
			slog.Error("Failed to stop tool sets", "error", err)
		}
	}()

	tracer := otel.Tracer(APP_NAME)

	rt, err := runtime.New(agents,
		runtime.WithCurrentAgent(agentName),
		runtime.WithAutoRunTools(autoApprove),
		runtime.WithTracer(tracer),
	)
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}

	sess := session.New()
	sess.Title = "Running agent"

	return runWithoutTUI(ctx, agentFilename, rt, sess, []string{"exec", "Follow the default instructions"})
}

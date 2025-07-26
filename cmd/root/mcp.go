package root

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/docker/cagent/pkg/servicecore"
)

var (
	agentsDir      string
	maxSessions    int
	sessionTimeout time.Duration
)

// NewMCPCmd creates the MCP server command
func NewMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp run",
		Short: "Start cagent in MCP server mode",
		Long: `Start cagent as an MCP (Model Context Protocol) server.
This allows external clients like Claude Code to programmatically invoke cagent agents
and maintain conversational sessions.`,
		RunE: runMCPCommand,
	}

	cmd.Flags().StringVar(&agentsDir, "agents-dir", "", "Directory containing agent configs (defaults to current directory)")
	cmd.Flags().IntVar(&maxSessions, "max-sessions", 100, "Maximum concurrent sessions")
	cmd.Flags().DurationVar(&sessionTimeout, "session-timeout", time.Hour, "Session timeout duration")
	cmd.Flags().BoolVar(&debugMode, "debug", false, "Enable debug logging")

	return cmd
}

func runMCPCommand(cmd *cobra.Command, args []string) error {
	// TODO: Initialize logger with appropriate level
	logger := slog.Default()
	if debugMode {
		logger.Info("Debug mode enabled")
	}

	// Default agents directory to current working directory if not specified
	resolvedAgentsDir := agentsDir
	if resolvedAgentsDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current working directory: %w", err)
		}
		resolvedAgentsDir = cwd
		logger.Info("Using current directory as agents root", "path", resolvedAgentsDir)
	}

	// Create servicecore manager
	serviceCore, err := servicecore.NewManager(resolvedAgentsDir, sessionTimeout, maxSessions, logger)
	if err != nil {
		return err
	}

	// TODO: Create MCP server using servicecore
	// mcpServer := mcpserver.NewMCPServer(serviceCore, logger)

	// TODO: Start MCP server
	// return mcpServer.Start(ctx)

	logger.Info("MCP server starting", "agents_dir", agentsDir, "max_sessions", maxSessions, "timeout", sessionTimeout)
	
	// Placeholder implementation - servicecore created successfully
	_ = serviceCore
	return nil
}
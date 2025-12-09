package root

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/mcp"
	"github.com/docker/cagent/pkg/server"
	"github.com/docker/cagent/pkg/telemetry"
)

type mcpFlags struct {
	agentName string
	http      bool
	port      int
	runConfig config.RuntimeConfig
}

func newMCPCmd() *cobra.Command {
	var flags mcpFlags

	cmd := &cobra.Command{
		Use:   "mcp <agent-file>|<registry-ref>",
		Short: "Start an agent as an MCP (Model Context Protocol) server",
		Long:  "Start an MCP server that exposes the agent via the Model Context Protocol. By default, uses stdio transport. Use --http to start a streaming HTTP server instead.",
		Example: `  cagent mcp ./agent.yaml
  cagent mcp ./team.yaml
  cagent mcp agentcatalog/pirate
  cagent mcp ./agent.yaml --http --port 8080`,
		Args:    cobra.ExactArgs(1),
		GroupID: "server",
		RunE:    flags.runMCPCommand,
	}

	cmd.PersistentFlags().StringVarP(&flags.agentName, "agent", "a", "", "Name of the agent to run (all agents if not specified)")
	cmd.PersistentFlags().BoolVar(&flags.http, "http", false, "Use streaming HTTP transport instead of stdio")
	cmd.PersistentFlags().IntVar(&flags.port, "port", 0, "Port to listen on when using HTTP transport (default: random available port)")
	addRuntimeConfigFlags(cmd, &flags.runConfig)

	return cmd
}

func (f *mcpFlags) runMCPCommand(cmd *cobra.Command, args []string) error {
	telemetry.TrackCommand("mcp", args)

	ctx := cmd.Context()
	agentFilename := args[0]

	if f.http || f.port != 0 {
		ln, err := server.Listen(ctx, fmt.Sprintf(":%d", f.port))
		if err != nil {
			return fmt.Errorf("failed to bind to port %d: %w", f.port, err)
		}
		go func() {
			<-ctx.Done()
			_ = ln.Close()
		}()

		return mcp.StartHTTPServer(ctx, agentFilename, f.agentName, &f.runConfig, ln)
	}

	return mcp.StartMCPServer(ctx, agentFilename, f.agentName, &f.runConfig)
}

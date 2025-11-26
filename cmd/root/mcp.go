package root

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/docker/cagent/pkg/agentfile"
	"github.com/docker/cagent/pkg/cli"
	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/mcp"
	"github.com/docker/cagent/pkg/telemetry"
)

type mcpFlags struct {
	runConfig config.RuntimeConfig
}

func newMCPCmd() *cobra.Command {
	var flags mcpFlags

	cmd := &cobra.Command{
		Use:   "mcp <agent-file>|<registry-ref>",
		Short: "Start an agent as an MCP (Model Context Protocol) server",
		Long:  "Start an stdio MCP server that exposes the agent via the Model Context Protocol",
		Example: `  cagent mcp ./agent.yaml
  cagent mcp ./team.yaml
  cagent mcp agentcatalog/pirate`,
		Args:    cobra.ExactArgs(1),
		GroupID: "server",
		RunE:    flags.runMCPCommand,
	}

	addRuntimeConfigFlags(cmd, &flags.runConfig)

	return cmd
}

func (f *mcpFlags) runMCPCommand(cmd *cobra.Command, args []string) error {
	telemetry.TrackCommand("mcp", args)

	ctx := cmd.Context()
	agentFilename := args[0]
	out := cli.NewPrinter(io.Discard)

	agentFilename, err := agentfile.Resolve(ctx, out, agentFilename)
	if err != nil {
		return err
	}

	return mcp.StartMCPServer(ctx, out, agentFilename, &f.runConfig)
}

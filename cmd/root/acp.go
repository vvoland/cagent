package root

import (
	"github.com/spf13/cobra"

	"github.com/docker/cagent/pkg/acp"
	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/telemetry"
)

type acpFlags struct {
	runConfig config.RuntimeConfig
}

func newACPCmd() *cobra.Command {
	var flags acpFlags

	cmd := &cobra.Command{
		Use:   "acp <agent-file>|<registry-ref>",
		Short: "Start an agent as an ACP (Agent Client Protocol) server",
		Long:  "Start an ACP server that exposes the agent via the Agent Client Protocol",
		Example: `  cagent acp ./agent.yaml
  cagent acp ./team.yaml
  cagent acp agentcatalog/pirate`,
		Args:    cobra.ExactArgs(1),
		GroupID: "server",
		RunE:    flags.runACPCommand,
	}

	addRuntimeConfigFlags(cmd, &flags.runConfig)

	return cmd
}

func (f *acpFlags) runACPCommand(cmd *cobra.Command, args []string) error {
	telemetry.TrackCommand("acp", args)

	ctx := cmd.Context()
	agentFilename := args[0]

	return acp.Run(ctx, agentFilename, cmd.InOrStdin(), cmd.OutOrStdout(), &f.runConfig)
}

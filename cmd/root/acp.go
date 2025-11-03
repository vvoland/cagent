package root

import (
	"log/slog"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/spf13/cobra"

	"github.com/docker/cagent/pkg/acp"
	"github.com/docker/cagent/pkg/telemetry"
)

func newACPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "acp <agent-file>",
		Short: "Start an ACP (Agent Client Protocol) server",
		Long:  `Start an ACP server that exposes the agent via the Agent Client Protocol`,
		Args:  cobra.ExactArgs(1),
		RunE:  runACPCommand,
	}

	addGatewayFlags(cmd)
	addRuntimeConfigFlags(cmd)

	return cmd
}

func runACPCommand(cmd *cobra.Command, args []string) error {
	telemetry.TrackCommand("acp", args)

	ctx := cmd.Context()
	agentFilename := args[0]

	slog.Debug("Starting ACP server", "agent_file", agentFilename, "debug_mode", debugMode)

	acpAgent := acp.NewAgent(agentFilename, runConfig)
	conn := acpsdk.NewAgentSideConnection(acpAgent, cmd.OutOrStdout(), cmd.InOrStdin())
	conn.SetLogger(slog.Default())
	acpAgent.SetAgentConnection(conn)
	defer acpAgent.Stop(ctx)

	slog.Debug("acp started, waiting for conn")

	<-conn.Done()

	return nil
}

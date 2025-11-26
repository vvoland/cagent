package root

import (
	"io"
	"log/slog"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/spf13/cobra"

	"github.com/docker/cagent/pkg/acp"
	"github.com/docker/cagent/pkg/agentfile"
	"github.com/docker/cagent/pkg/cli"
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
	out := cli.NewPrinter(io.Discard)

	agentFilename, err := agentfile.Resolve(ctx, out, agentFilename)
	if err != nil {
		return err
	}

	slog.Debug("Starting ACP server", "agent_file", agentFilename)

	acpAgent := acp.NewAgent(agentFilename, &f.runConfig)
	conn := acpsdk.NewAgentSideConnection(acpAgent, cmd.OutOrStdout(), cmd.InOrStdin())
	conn.SetLogger(slog.Default())
	acpAgent.SetAgentConnection(conn)
	defer acpAgent.Stop(ctx)

	slog.Debug("acp started, waiting for conn")
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-conn.Done():
		return nil
	}
}

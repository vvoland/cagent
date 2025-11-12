package root

import (
	"github.com/spf13/cobra"

	"github.com/docker/cagent/pkg/cli"
	"github.com/docker/cagent/pkg/telemetry"
)

func newExecCmd() *cobra.Command {
	var flags runExecFlags

	cmd := &cobra.Command{
		Use:   "exec <agent-name>",
		Short: "Execute an agent",
		Long:  "Execute an agent (Single user message / No TUI)",
		Example: `  cagent exec ./agent.yaml
  cagent exec ./team.yaml --agent root
  cagent exec ./echo.yaml "INSTRUCTIONS"
  echo "INSTRUCTIONS" | cagent exec ./echo.yaml -`,
		Args: cobra.RangeArgs(1, 2),
		RunE: flags.runExecCommand,
	}

	addRunOrExecFlags(cmd, &flags)
	addRuntimeConfigFlags(cmd, &flags.runConfig)

	return cmd
}

func (f *runExecFlags) runExecCommand(cmd *cobra.Command, args []string) error {
	telemetry.TrackCommand("exec", args)

	ctx := cmd.Context()
	out := cli.NewPrinter(cmd.OutOrStdout())

	return f.runOrExec(ctx, out, args, true)
}

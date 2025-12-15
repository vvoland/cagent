package root

import (
	"github.com/spf13/cobra"

	"github.com/docker/cagent/pkg/cli"
	"github.com/docker/cagent/pkg/telemetry"
)

func newExecCmd() *cobra.Command {
	var flags runExecFlags

	cmd := &cobra.Command{
		Use:   "exec <agent-file>|<registry-ref>",
		Short: "Execute an agent",
		Long:  "Execute an agent (Single user message / No TUI)",
		Example: `  cagent exec ./agent.yaml
  cagent exec ./team.yaml --agent root
  cagent exec ./echo.yaml "INSTRUCTIONS"
  echo "INSTRUCTIONS" | cagent exec ./echo.yaml -
  cagent exec ./agent.yaml "question" --record  # Records to auto-generated file
  cagent exec ./agent.yaml "question" --record session.yaml  # Records to specific file`,
		GroupID: "core",
		Args:    cobra.RangeArgs(1, 2),
		RunE:    flags.runExecCommand,
	}

	addRunOrExecFlags(cmd, &flags)
	addRuntimeConfigFlags(cmd, &flags.runConfig)
	cmd.PersistentFlags().BoolVar(&flags.hideToolCalls, "hide-tool-calls", false, "Hide the tool calls in the output")
	cmd.PersistentFlags().BoolVar(&flags.outputJSON, "json", false, "Output results in JSON format")
	cmd.PersistentFlags().StringVar(&flags.recordPath, "record", "", "Record AI API interactions to cassette file (auto-generates filename if empty)")
	cmd.PersistentFlags().Lookup("record").NoOptDefVal = "true" // Allow --record without value

	return cmd
}

func (f *runExecFlags) runExecCommand(cmd *cobra.Command, args []string) error {
	telemetry.TrackCommand("exec", args)

	ctx := cmd.Context()
	out := cli.NewPrinter(cmd.OutOrStdout())

	tui := false
	return f.runOrExec(ctx, out, args, tui)
}

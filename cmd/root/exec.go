package root

import (
	"github.com/spf13/cobra"

	"github.com/docker/cagent/pkg/cli"
	"github.com/docker/cagent/pkg/telemetry"
)

func NewExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec <agent-name>",
		Short: "Execute an agent",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			out := cli.NewPrinter(cmd.OutOrStdout())

			telemetry.TrackCommand("exec", args)
			setupOtel(ctx)

			return runOrExec(ctx, out, args, true)
		},
	}

	cmd.PersistentFlags().StringVarP(&agentName, "agent", "a", "root", "Name of the agent to run")
	cmd.PersistentFlags().StringVar(&workingDir, "working-dir", "", "Set the working directory for the session (applies to tools and relative paths)")
	cmd.PersistentFlags().BoolVar(&autoApprove, "yolo", false, "Automatically approve all tool calls without prompting")
	cmd.PersistentFlags().StringVar(&attachmentPath, "attach", "", "Attach an image file to the message")
	cmd.PersistentFlags().StringArrayVar(&modelOverrides, "model", nil, "Override agent model: [agent=]provider/model (repeatable)")
	cmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Initialize the agent without executing anything")
	_ = cmd.PersistentFlags().MarkHidden("dry-run")

	addGatewayFlags(cmd)
	addRuntimeConfigFlags(cmd)

	return cmd
}

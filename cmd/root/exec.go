package root

import (
	"context"

	"github.com/docker/cagent/pkg/telemetry"
	"github.com/spf13/cobra"
)

func NewExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec <agent-name>",
		Short: "Execute an agent",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return execCommand(cmd.Context(), args)
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

func execCommand(ctx context.Context, args []string) error {
	telemetry.TrackCommand("exec", args)
	setupOtel(ctx)
	return doRunCommand(ctx, args, true)
}

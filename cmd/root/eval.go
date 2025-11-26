package root

import (
	"github.com/spf13/cobra"

	"github.com/docker/cagent/pkg/cli"
	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/evaluation"
	"github.com/docker/cagent/pkg/telemetry"
)

type evalFlags struct {
	runConfig config.RuntimeConfig
}

func newEvalCmd() *cobra.Command {
	var flags evalFlags

	cmd := &cobra.Command{
		Use:     "eval <agent-file>|<registry-ref> [<eval-dir>|./evals]",
		Short:   "Run evaluations for an agent",
		GroupID: "advanced",
		Args:    cobra.RangeArgs(1, 2),
		RunE:    flags.runEvalCommand,
	}

	addRuntimeConfigFlags(cmd, &flags.runConfig)

	return cmd
}

func (f *evalFlags) runEvalCommand(cmd *cobra.Command, args []string) error {
	telemetry.TrackCommand("eval", args)

	ctx := cmd.Context()
	out := cli.NewPrinter(cmd.OutOrStdout())
	agentFilename := args[0]
	evalsDir := "./evals"
	if len(args) >= 2 {
		evalsDir = args[1]
	}

	return evaluation.Evaluate(ctx, out, agentFilename, evalsDir, &f.runConfig)
}

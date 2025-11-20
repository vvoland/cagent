package root

import (
	"github.com/spf13/cobra"

	"github.com/docker/cagent/pkg/agentfile"
	"github.com/docker/cagent/pkg/cli"
	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/evaluation"
	"github.com/docker/cagent/pkg/teamloader"
	"github.com/docker/cagent/pkg/telemetry"
)

type evalFlags struct {
	runConfig config.RuntimeConfig
}

func newEvalCmd() *cobra.Command {
	var flags evalFlags

	cmd := &cobra.Command{
		Use:     "eval <agent-file>|<registry-ref> <eval-dir>",
		Short:   "Run evaluations for an agent",
		GroupID: "advanced",
		Args:    cobra.ExactArgs(2),
		RunE:    flags.runEvalCommand,
	}

	addRuntimeConfigFlags(cmd, &flags.runConfig)

	return cmd
}

func (f *evalFlags) runEvalCommand(cmd *cobra.Command, args []string) error {
	telemetry.TrackCommand("eval", args)

	ctx := cmd.Context()
	out := cli.NewPrinter(cmd.OutOrStdout())

	agentFilename, err := agentfile.Resolve(ctx, out, args[0])
	if err != nil {
		return err
	}

	agents, err := teamloader.Load(ctx, agentFilename, &f.runConfig)
	if err != nil {
		return err
	}

	evalResults, err := evaluation.Evaluate(ctx, agents, args[1])
	if err != nil {
		return err
	}

	for _, evalResult := range evalResults {
		out.Printf("Eval file: %s\n", evalResult.EvalFile)
		out.Printf("Tool trajectory score: %f\n", evalResult.Score.ToolTrajectoryScore)
		out.Printf("Rouge-1 score: %f\n", evalResult.Score.Rouge1Score)
	}

	return nil
}

package root

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/docker/cagent/pkg/evaluation"
	"github.com/docker/cagent/pkg/loader"
)

func NewEvalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "eval <agent-name> <eval-dir>",
		Args: cobra.ExactArgs(2),
		RunE: runEvalCommand,
	}

	cmd.PersistentFlags().StringSliceVar(&runConfig.EnvFiles, "env-from-file", nil, "Set environment variables from file")
	addGatewayFlags(cmd)

	return cmd
}

func runEvalCommand(cmd *cobra.Command, args []string) error {
	logger := slog.Default()

	agents, err := loader.Load(cmd.Context(), args[0], runConfig, logger)
	if err != nil {
		return err
	}

	evalResults, err := evaluation.Evaluate(cmd.Context(), agents, args[1], logger)
	if err != nil {
		return err
	}

	for _, evalResult := range evalResults {
		fmt.Printf("Eval file: %s\n", evalResult.EvalFile)
		fmt.Printf("Tool trajectory score: %f\n", evalResult.Score.ToolTrajectoryScore)
		fmt.Printf("Rouge-1 score: %f\n", evalResult.Score.Rouge1Score)
	}

	return nil
}

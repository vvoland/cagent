package root

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/docker/cagent/pkg/teamloader"
	"github.com/docker/cagent/pkg/telemetry"
)

func newDebugCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "debug",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "toolsets <agent-name>",
		Short: "Debug the toolsets of an agent",
		Args:  cobra.ExactArgs(1),
		RunE:  runDebugToolsetsCommand,
	})

	return cmd
}

func runDebugToolsetsCommand(cmd *cobra.Command, args []string) error {
	telemetry.TrackCommand("debug", append([]string{"toolsets"}, args...))

	ctx := cmd.Context()
	agentFilename := args[0]

	slog.Info("Loading agent", "agent", agentFilename)
	team, err := teamloader.Load(ctx, agentFilename, runConfig)
	if err != nil {
		return err
	}

	for _, name := range team.AgentNames() {
		agent, err := team.Agent(name)
		if err != nil {
			slog.Error("Failed to get agent", "name", name, "error", err)
			continue
		}

		slog.Info("Query tools", "name", agent.Name())
		tools, err := agent.Tools(ctx)
		if err != nil {
			slog.Error("Failed to query tools", "name", agent.Name(), "error", err)
			continue
		}

		for _, tool := range tools {
			slog.Info("Tool found", "name", tool.Name)
		}
	}

	slog.Info("Stopping toolsets", "agent", agentFilename)
	if err := team.StopToolSets(ctx); err != nil {
		slog.Error("Failed to stop tool sets", "error", err)
	}

	return err
}

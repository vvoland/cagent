package root

import (
	"github.com/docker/cagent/pkg/teamloader"
	"github.com/spf13/cobra"
)

// NewDebugCmd creates a command that prints the debug information about cagent.
func NewDebugCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "debug",
		Hidden: true,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "toolsets <agent-name>",
		Short: "Debug the toolsets of an agent",
		Args:  cobra.ExactArgs(1),
		RunE:  debugToolsetsCommand,
	})

	return cmd
}

func debugToolsetsCommand(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	agentFilename := args[0]

	logger := newLogger()

	logger.Info("Loading agent", "agent", agentFilename)
	team, err := teamloader.Load(ctx, agentFilename, runConfig, logger)
	if err != nil {
		return err
	}

	for _, name := range team.AgentNames() {
		agent := team.Agent(name)
		logger.Info("Query tools", "name", agent.Name())
		tools, err := agent.Tools(ctx)
		if err != nil {
			logger.Error("Failed to query tools", "name", agent.Name(), "error", err)
			continue
		}

		for _, tool := range tools {
			logger.Info("Tool found", "name", tool.Function.Name)
		}
	}

	logger.Info("Stopping toolsets", "agent", agentFilename)
	if err := team.StopToolSets(); err != nil {
		logger.Error("Failed to stop tool sets", "error", err)
	}

	return err
}

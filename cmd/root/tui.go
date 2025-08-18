package root

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/docker/cagent/internal/tui"
	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/teamloader"
)

// NewTUICmd creates a new TUI command
func NewTUICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tui <agent-name>",
		Short: "Run the agent with a TUI",
		Long:  `Run the agent with a Terminal User Interface powered by Charm`,
		Args:  cobra.ExactArgs(1),
		RunE:  runTUICommand,
	}

	cmd.PersistentFlags().StringVarP(&agentName, "agent", "a", "root", "Name of the agent to run")
	cmd.PersistentFlags().StringSliceVar(&runConfig.EnvFiles, "env-from-file", nil, "Set environment variables from file")
	addGatewayFlags(cmd)

	return cmd
}

func runTUICommand(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	agentFilename := args[0]

	logger := newLogger()

	logger.Debug("Starting agent TUI", "agent", agentName, "debug_mode", debugMode)

	agents, err := teamloader.Load(ctx, agentFilename, runConfig, logger)
	if err != nil {
		return err
	}
	defer func() {
		if err := agents.StopToolSets(); err != nil {
			logger.Error("Failed to stop tool sets", "error", err)
		}
	}()

	rt := runtime.New(logger, agents, runtime.WithCurrentAgent(agentName))

	m, err := tui.NewModel(rt, session.New(logger))
	if err != nil {
		return err
	}

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseAllMotion(), // Enable mouse support
		tea.WithMouseCellMotion(),
	)

	_, err = p.Run()
	return err
}

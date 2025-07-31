package root

import (
	"fmt"

	"github.com/docker/cagent/pkg/config"
	"github.com/spf13/cobra"
)

// NewReadmeCmd creates a command that prints the README of an agent.
func NewReadmeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "readme <agent-name>",
		Short: "Print the README of an agent",
		Args:  cobra.ExactArgs(1),
		RunE:  readmeAgentCommand,
	}

	return cmd
}

func readmeAgentCommand(cmd *cobra.Command, args []string) error {
	agentFilename := args[0]

	cfg, err := config.LoadConfig(agentFilename)
	if err != nil {
		return err
	}

	_, err = fmt.Print(cfg.Readme)
	return err
}

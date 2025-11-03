package root

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/filesystem"
	"github.com/docker/cagent/pkg/telemetry"
)

func newReadmeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "readme <agent-name>",
		Short: "Print the README of an agent",
		Args:  cobra.ExactArgs(1),
		RunE:  readmeReadmeCommand,
	}
}

func readmeReadmeCommand(_ *cobra.Command, args []string) error {
	telemetry.TrackCommand("readme", args)

	agentFilename := args[0]

	cfg, err := config.LoadConfig(agentFilename, filesystem.AllowAll)
	if err != nil {
		return err
	}

	_, err = fmt.Print(cfg.Metadata.Readme)
	return err
}

package root

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/docker/cagent/pkg/telemetry"
	"github.com/docker/cagent/pkg/version"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Long:  `Display the version, build time, and commit hash`,
		Args:  cobra.NoArgs,
		Run:   runVersionCommand,
	}
}

func runVersionCommand(_ *cobra.Command, args []string) {
	telemetry.TrackCommand("version", args)

	fmt.Printf("cagent version %s\n", version.Version)
	fmt.Printf("Commit: %s\n", version.Commit)
}

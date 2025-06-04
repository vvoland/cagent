package root

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version information
var (
	Version   = "dev"
	BuildTime = "unknown"
	Commit    = "unknown"
)

// NewVersionCmd creates a new version command
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Long:  `Display the version, build time, and commit hash`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("cagent version %s\n", Version)
			fmt.Printf("Build time: %s\n", BuildTime)
			fmt.Printf("Commit: %s\n", Commit)
		},
	}
}

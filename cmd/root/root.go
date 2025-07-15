package root

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	agentName string
	debugMode bool
)

// NewRootCmd creates the root command for cagent
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cagent",
		Short: "cagent - AI agent runner",
		Long:  `cagent is a command-line tool for running AI agents`,
		// If no subcommand is specified, show help
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(NewVersionCmd())
	cmd.AddCommand(NewRunCmd())
	cmd.AddCommand(NewWebCmd())
	cmd.AddCommand(NewUICmd())
	cmd.AddCommand(NewInitCmd())
	cmd.AddCommand(NewApiCmd())

	return cmd
}

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

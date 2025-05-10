package root

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	configFile    string
	agentName     string
	initialPrompt string
)

// NewRootCmd creates the root command for cagent
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cagent",
		Short: "cagent - AI agent runner",
		Long:  `cagent is a command-line tool for running AI agents`,
		// If no subcommand is specified, show help
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// Add flags
	cmd.PersistentFlags().StringVarP(&configFile, "config", "c", "agent.yaml", "Path to the configuration file")
	cmd.PersistentFlags().StringVarP(&agentName, "agent", "a", "root", "Name of the agent to run")
	cmd.PersistentFlags().StringVarP(&initialPrompt, "prompt", "p", "", "Initial prompt to send to the agent")

	// Add subcommands
	cmd.AddCommand(NewVersionCmd())
	cmd.AddCommand(NewRunCmd())

	return cmd
}

// Execute runs the root command
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

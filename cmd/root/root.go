package root

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var (
	agentName  string
	debugMode  bool
	enableOtel bool
)

// NewRootCmd creates the root command for cagent
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cagent",
		Short: "cagent - AI agent runner",
		Long:  `cagent is a command-line tool for running AI agents`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.DisplayName() != "exec" {
				_, _ = cmd.OutOrStdout().Write([]byte("\nFor any feedback, please visit: " + FeedbackLink + "\n\n"))
			}

			if debugMode {
				slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
					Level: slog.LevelDebug,
				})))
			}
			return nil
		},
		// If no subcommand is specified, show help
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add persistent debug flag available to all commands
	cmd.PersistentFlags().BoolVarP(&debugMode, "debug", "d", false, "Enable debug logging")
	cmd.PersistentFlags().BoolVarP(&enableOtel, "otel", "o", false, "Enable OpenTelemetry tracing")

	cmd.AddCommand(NewVersionCmd())
	cmd.AddCommand(NewRunCmd())
	cmd.AddCommand(NewExecCmd())
	cmd.AddCommand(NewTuiCmd())
	cmd.AddCommand(NewNewCmd())
	cmd.AddCommand(NewApiCmd())
	cmd.AddCommand(NewEvalCmd())
	cmd.AddCommand(NewPushCmd())
	cmd.AddCommand(NewPullCmd())
	cmd.AddCommand(NewMCPCmd())
	cmd.AddCommand(NewReadmeCmd())
	cmd.AddCommand(NewDebugCmd())
	cmd.AddCommand(NewFeedbackCmd())
	cmd.AddCommand(NewCatalogCmd())

	return cmd
}

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

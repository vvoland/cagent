package root

import (
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"

	"github.com/docker/cagent/pkg/cli"
	"github.com/docker/cagent/pkg/telemetry"
	"github.com/docker/cagent/pkg/userconfig"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage user configuration",
		Long:  "View and manage user-level cagent configuration stored in ~/.config/cagent/config.yaml",
		Example: `  # Show the current configuration
  cagent config show

  # Show the path to the config file
  cagent config path`,
		GroupID: "advanced",
		RunE:    runConfigShowCommand,
	}

	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newConfigPathCmd())

	return cmd
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show the current configuration",
		Long:  "Display the current user configuration in YAML format",
		Args:  cobra.NoArgs,
		RunE:  runConfigShowCommand,
	}
}

func newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Show the path to the config file",
		Args:  cobra.NoArgs,
		RunE:  runConfigPathCommand,
	}
}

func runConfigShowCommand(cmd *cobra.Command, _ []string) error {
	telemetry.TrackCommand("config", []string{"show"})

	out := cli.NewPrinter(cmd.OutOrStdout())

	config, err := userconfig.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	data, err := yaml.MarshalWithOptions(config, yaml.IndentSequence(true), yaml.UseSingleQuote(false))
	if err != nil {
		return fmt.Errorf("failed to format config: %w", err)
	}

	out.Print(string(data))
	return nil
}

func runConfigPathCommand(cmd *cobra.Command, _ []string) error {
	telemetry.TrackCommand("config", []string{"path"})

	out := cli.NewPrinter(cmd.OutOrStdout())
	out.Println(userconfig.Path())
	return nil
}

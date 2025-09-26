package root

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"

	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/telemetry"
)

func NewPrintCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "print <agent-name>",
		Short:  "Print the canonical configuration of an agent",
		Args:   cobra.ExactArgs(1),
		RunE:   printCommand,
		Hidden: true,
	}
}

func printCommand(cmd *cobra.Command, args []string) error {
	telemetry.TrackCommand("print", args)

	agentFilename := args[0]

	ext := strings.ToLower(filepath.Ext(agentFilename))
	if ext == ".yaml" || ext == ".yml" || strings.HasPrefix(agentFilename, "/dev/fd/") {
		cfg, err := config.LoadConfigSecure(filepath.Base(agentFilename), filepath.Dir(agentFilename))
		if err != nil {
			return err
		}

		buf, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}

		fmt.Println(string(buf))
	}

	return nil
}

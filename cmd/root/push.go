package root

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/docker/cagent/pkg/remote"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/spf13/cobra"
)

func NewPushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push <reference>",
		Short: "Push an artifact to an OCI registry",
		Long: `Push an artifact from the local content store to an OCI registry.

The local identifier can be either a reference (tag) or a digest that was returned
from the build command.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPushCommand(args[0])
		},
	}

	return cmd
}

func runPushCommand(reference string) error {
	logLevel := slog.LevelInfo
	if debugMode {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))

	logger.Debug("Starting push", "registry_ref", reference)

	var opts []crane.Option

	err := remote.Push(reference, opts...)
	if err != nil {
		return fmt.Errorf("failed to push artifact: %w", err)
	}

	fmt.Printf("Successfully pushed artifact to %s\n", reference)
	return nil
}

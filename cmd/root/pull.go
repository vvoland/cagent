package root

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/docker/cagent/pkg/remote"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/spf13/cobra"
)

func NewPullCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull <registry-ref>",
		Short: "Pull an artifact from Docker Hub",
		Long:  `Pull an artifact from Docker Hub`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPullCommand(args[0])
		},
	}

	return cmd
}

func runPullCommand(registryRef string) error {
	logLevel := slog.LevelInfo
	if debugMode {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))

	logger.Debug("Starting pull", "registry_ref", registryRef)

	var opts []crane.Option
	digest, err := remote.Pull(registryRef, opts...)
	if err != nil {
		return fmt.Errorf("failed to pull artifact: %w", err)
	}
	fmt.Printf("Successfully pulled artifact from %s (digest: %s)\n", registryRef, digest)

	return nil
}

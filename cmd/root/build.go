package root

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/docker/cagent/pkg/content"
	"github.com/docker/cagent/pkg/oci"
	"github.com/spf13/cobra"
)

func NewBuildCmd() *cobra.Command {
	var (
		debug bool
		tag   string
	)

	cmd := &cobra.Command{
		Use:   "build [flags] <file>",
		Short: "Build an artifact from a file",
		Long: `Build an OCI artifact from a single file.

The command packages the specified file as an OCI artifact and stores it 
in the local content store with the given tag.

Examples:
  cagent build -t my-app:v1.0.0 config.yaml
  cagent build -t rumpl/toto agent.yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if tag == "" {
				return fmt.Errorf("tag is required, use -t or --tag to specify it")
			}
			return runBuildCommand(args[0], tag, debug)
		},
	}

	cmd.Flags().StringVarP(&tag, "tag", "t", "", "Tag for the artifact (required)")
	cmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug output")
	_ = cmd.MarkFlagRequired("tag")

	return cmd
}

func runBuildCommand(filePath, tag string, debug bool) error {
	logLevel := slog.LevelInfo
	if debug {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))

	logger.Debug("Starting build", "file", filePath, "tag", tag)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", filePath)
	}

	store, err := content.NewStore()
	if err != nil {
		return err
	}

	digest, err := oci.PackageFileAsOCIToStore(filePath, tag, store)
	if err != nil {
		return fmt.Errorf("failed to build artifact: %w", err)
	}

	fmt.Printf("Successfully built artifact %s (digest: %s)\n", tag, digest)
	return nil
}

package agentfile

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"

	"github.com/docker/cagent/pkg/aliases"
	"github.com/docker/cagent/pkg/cli"
	"github.com/docker/cagent/pkg/content"
	"github.com/docker/cagent/pkg/remote"
)

// IsOCIReference checks if the input is a valid OCI reference
func IsOCIReference(input string) bool {
	if IsLocalFile(input) {
		return false
	}
	_, err := name.ParseReference(input)
	return err == nil
}

// IsLocalFile checks if the input is a local file
func IsLocalFile(input string) bool {
	ext := strings.ToLower(filepath.Ext(input))
	// Check for YAML file extensions or file descriptors
	if ext == ".yaml" || ext == ".yml" || strings.HasPrefix(input, "/dev/fd/") {
		return true
	}
	// Check if it exists as a file on disk
	return fileExists(input)
}

// OciRefToFilename converts an OCI reference to a safe, consistent filename
// Examples:
//   - "docker.io/myorg/agent:v1" -> "docker.io_myorg_agent_v1.yaml"
//   - "localhost:5000/test" -> "localhost_5000_test.yaml"
func OciRefToFilename(ociRef string) string {
	// Replace characters that are invalid in filenames with underscores
	// Keep the structure recognizable but filesystem-safe
	safe := strings.NewReplacer(
		"/", "_",
		":", "_",
		"@", "_",
		"\\", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	).Replace(ociRef)

	// Ensure it has .yaml extension
	if !strings.HasSuffix(safe, ".yaml") {
		safe += ".yaml"
	}

	return safe
}

// Resolve resolves an agent file reference (local file or OCI image) to a local file path
// For OCI references, always checks remote for updates but falls back to local cache if offline
func Resolve(ctx context.Context, out *cli.Printer, agentFilename string) (string, error) {
	originalOCIRef := agentFilename // Store the original for OCI ref tracking

	if agentFilename == "" {
		agentFilename = "default"
	}

	// Try to resolve as an alias first
	if aliasStore, err := aliases.Load(); err == nil {
		if resolvedPath, ok := aliasStore.Get(agentFilename); ok {
			slog.Debug("Resolved alias", "alias", agentFilename, "path", resolvedPath)
			agentFilename = resolvedPath
		}
	}

	// "default" is either a user defined alias or the default (embedded) agent
	if agentFilename == "default" {
		return "default", nil
	}

	if IsLocalFile(agentFilename) {
		// Treat as local YAML file: resolve to absolute path so later chdir doesn't break it
		// TODO(rumpl): Why are we checking for newlines here?
		if !strings.Contains(agentFilename, "\n") {
			if abs, err := filepath.Abs(agentFilename); err == nil {
				agentFilename = abs
			}
		}
		if !fileExists(agentFilename) {
			return "", fmt.Errorf("agent file not found: %s", agentFilename)
		}
		return agentFilename, nil
	}

	// Treat as an OCI image reference
	// Check if we have a local copy (without loading content)
	store, err := content.NewStore()
	if err != nil {
		return "", fmt.Errorf("failed to create content store: %w", err)
	}

	_, metaErr := store.GetArtifactMetadata(agentFilename)
	hasLocal := metaErr == nil

	if out != nil {
		if hasLocal {
			out.Printf("Checking for updates to OCI reference %s...\n", agentFilename)
		} else {
			out.Printf("Pulling OCI reference %s...\n", agentFilename)
		}
	}

	// Try to pull from remote (only pulls if digest changed)
	if _, pullErr := remote.Pull(ctx, agentFilename, false); pullErr != nil {
		if !hasLocal {
			// No local copy and can't pull, error out
			return "", fmt.Errorf("failed to pull OCI image %s: %w", agentFilename, pullErr)
		}
		slog.Debug("Failed to check for OCI reference updates, using cached version", "ref", agentFilename, "error", pullErr)
		if out != nil {
			out.Println("Using cached version of", agentFilename)
		}
	}

	// Load the agent contents from the store
	a, err := FromStore(agentFilename)
	if err != nil {
		return "", fmt.Errorf("failed to load agent from store: %w", err)
	}

	filename := OciRefToFilename(originalOCIRef)
	tmpFilename := filepath.Join(os.TempDir(), filename)

	if err := os.WriteFile(tmpFilename, []byte(a), 0o644); err != nil {
		return "", fmt.Errorf("failed to write agent file: %w", err)
	}

	slog.Debug("Resolved OCI reference to file", "oci_ref", originalOCIRef, "file", tmpFilename)

	go func() {
		<-ctx.Done()
		os.Remove(tmpFilename)
		slog.Debug("Cleaned up OCI reference file", "file", tmpFilename)
	}()

	return tmpFilename, nil
}

// fileExists checks if a file exists at the given path
func fileExists(path string) bool {
	_, err := os.Stat(path)
	exists := err == nil
	return exists
}

// FromStore loads an agent configuration from the OCI content store
func FromStore(reference string) (string, error) {
	store, err := content.NewStore()
	if err != nil {
		return "", err
	}

	img, err := store.GetArtifactImage(reference)
	if err != nil {
		return "", err
	}

	layers, err := img.Layers()
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	layer := layers[0]
	b, err := layer.Uncompressed()
	if err != nil {
		return "", err
	}

	_, err = io.Copy(&buf, b)
	if err != nil {
		return "", err
	}
	b.Close()

	return buf.String(), nil
}

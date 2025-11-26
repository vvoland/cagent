package agentfile

import (
	"context"
	_ "embed"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"

	"github.com/docker/cagent/pkg/aliases"
	"github.com/docker/cagent/pkg/teamloader"
)

//go:embed default-agent.yaml
var defaultAgent []byte

type Printer interface {
	Printf(format string, a ...any)
}

// ResolveSource resolves an agent file reference (local file or OCI image) to a local file path
// For OCI references, always checks remote for updates but falls back to local cache if offline
func ResolveSource(ctx context.Context, out Printer, agentFilename string) (teamloader.AgentSource, error) {
	resolvedPath, err := Resolve(ctx, out, agentFilename)
	if err != nil {
		return nil, err
	}

	if resolvedPath == "default" {
		return teamloader.NewBytesSource(defaultAgent), nil
	}
	if isLocalFile(resolvedPath) {
		return teamloader.NewFileSource(resolvedPath), nil
	}
	return teamloader.NewOCISource(resolvedPath), nil
}

// Resolve resolves an agent file reference (local file or OCI image) to a local file path
func Resolve(ctx context.Context, out Printer, agentFilename string) (string, error) {
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

	return agentFilename, nil
}

// fileExists checks if a file exists at the given path
func fileExists(path string) bool {
	_, err := os.Stat(path)
	exists := err == nil
	return exists
}

// IsOCIReference checks if the input is a valid OCI reference
func IsOCIReference(input string) bool {
	if isLocalFile(input) {
		return false
	}
	_, err := name.ParseReference(input)
	return err == nil
}

// isLocalFile checks if the input is a local file
func isLocalFile(input string) bool {
	ext := strings.ToLower(filepath.Ext(input))
	// Check for YAML file extensions or file descriptors
	if ext == ".yaml" || ext == ".yml" || strings.HasPrefix(input, "/dev/fd/") {
		return true
	}
	// Check if it exists as a file on disk
	return fileExists(input)
}

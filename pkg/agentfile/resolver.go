package agentfile

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"

	"github.com/docker/cagent/pkg/aliases"
	"github.com/docker/cagent/pkg/reference"
)

//go:embed default-agent.yaml
var defaultAgent []byte

type Printer interface {
	Printf(format string, a ...any)
}

// ResolveSource resolves an agent file reference (local file or OCI image) to a local file path
// For OCI references, always checks remote for updates but falls back to local cache if offline
func ResolveSources(ctx context.Context, agentFilename string) (AgentSources, error) {
	resolvedPath, err := resolve(agentFilename)
	if err != nil {
		if IsOCIReference(agentFilename) {
			return map[string]AgentSource{reference.OciRefToFilename(agentFilename): NewOCISource(agentFilename)}, nil
		}
		return nil, err
	}

	if resolvedPath == "default" {
		return map[string]AgentSource{"default": NewBytesSource(defaultAgent)}, nil
	}
	if isLocalFile(resolvedPath) {
		return map[string]AgentSource{resolvedPath: NewFileSource(resolvedPath)}, nil
	}
	if dirExists(resolvedPath) {
		sources := make(AgentSources)
		entries, err := os.ReadDir(resolvedPath)
		if err != nil {
			return nil, fmt.Errorf("reading agents directory %s: %w", resolvedPath, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext != ".yaml" && ext != ".yml" {
				continue
			}
			a := filepath.Join(resolvedPath, entry.Name())
			sources[a], err = Resolve(ctx, a)
			if err != nil {
				return nil, err
			}
		}
		return sources, nil
	}
	return map[string]AgentSource{resolvedPath: NewOCISource(agentFilename)}, nil
}

// Resolve resolves an agent file reference (local file or OCI image) to a local file path
// For OCI references, always checks remote for updates but falls back to local cache if offline
func Resolve(ctx context.Context, agentFilename string) (AgentSource, error) {
	resolvedPath, err := resolve(agentFilename)
	if err != nil {
		if IsOCIReference(agentFilename) {
			return NewOCISource(agentFilename), nil
		}
		return nil, err
	}

	if resolvedPath == "default" {
		return NewBytesSource(defaultAgent), nil
	}
	if isLocalFile(resolvedPath) {
		return NewFileSource(resolvedPath), nil
	}
	return NewOCISource(agentFilename), nil
}

// resolve resolves an agent reference, handling aliases and defaults
func resolve(agentFilename string) (string, error) {
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

	abs, err := filepath.Abs(agentFilename)
	if err != nil {
		return "", err
	}

	return abs, nil
}

// fileExists checks if a file exists at the given path
func fileExists(path string) bool {
	s, err := os.Stat(path)
	exists := err == nil && !s.IsDir()
	return exists
}

// fileExists checks if a file exists at the given path
func dirExists(path string) bool {
	s, err := os.Stat(path)
	exists := err == nil && s.IsDir()
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

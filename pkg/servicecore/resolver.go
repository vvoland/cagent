package servicecore

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/cagent/pkg/content"
	"github.com/docker/cagent/pkg/remote"
)

// Resolver handles agent specification resolution (file path, image reference, or store reference)
type Resolver struct {
	agentsDir string
	store     *content.Store
	logger    *slog.Logger
}

// NewResolver creates a new agent resolver
func NewResolver(agentsDir string, logger *slog.Logger) (*Resolver, error) {
	store, err := content.NewStore()
	if err != nil {
		return nil, fmt.Errorf("creating content store: %w", err)
	}

	return &Resolver{
		agentsDir: agentsDir,
		store:     store,
		logger:    logger,
	}, nil
}

// ResolveAgent resolves an agent specification to a file path
// Priority: File path → Content store → Error
func (r *Resolver) ResolveAgent(agentSpec string) (string, error) {
	r.logger.Debug("Resolving agent", "agent_spec", agentSpec)

	// Check if it's a file path that exists
	if r.fileExists(agentSpec) {
		r.logger.Debug("Agent resolved as file path", "path", agentSpec)
		return agentSpec, nil
	}

	// Check if it's a relative path in agents directory
	if !filepath.IsAbs(agentSpec) {
		fullPath := filepath.Join(r.agentsDir, agentSpec)
		if r.fileExists(fullPath) {
			r.logger.Debug("Agent resolved as relative path", "path", fullPath)
			return fullPath, nil
		}
	}

	// Try to load from content store (Docker images)
	yamlContent, err := r.fromStore(agentSpec)
	if err != nil {
		return "", fmt.Errorf("agent not found in files or store: %w", err)
	}

	// Create temporary file with YAML content
	tmpFile, err := os.CreateTemp("", "mcpagent-*.yaml")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}

	if _, err := tmpFile.WriteString(yamlContent); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("writing temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("closing temp file: %w", err)
	}

	r.logger.Debug("Agent resolved from store", "agent_spec", agentSpec, "temp_file", tmpFile.Name())
	return tmpFile.Name(), nil
}

// ListFileAgents lists agents available as files
func (r *Resolver) ListFileAgents() ([]AgentInfo, error) {
	agents := []AgentInfo{}

	// Expand tilde in agents directory
	agentsDir := r.expandPath(r.agentsDir)

	// Check if agents directory exists
	if !r.fileExists(agentsDir) {
		r.logger.Debug("Agents directory does not exist", "path", agentsDir)
		return agents, nil
	}

	// Walk through agents directory
	err := filepath.Walk(agentsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only include .yaml and .yml files
		if !info.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			relPath, err := filepath.Rel(agentsDir, path)
			if err != nil {
				relPath = path
			}

			agent := AgentInfo{
				Name:        strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
				Description: fmt.Sprintf("File-based agent: %s", relPath),
				Source:      "file",
				Path:        path,
			}
			agents = append(agents, agent)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("listing file agents: %w", err)
	}

	r.logger.Debug("Listed file agents", "count", len(agents))
	return agents, nil
}

// ListStoreAgents lists agents available in the content store
func (r *Resolver) ListStoreAgents() ([]AgentInfo, error) {
	// TODO: Implement store listing - this would require extending the content store
	// For now, return empty list
	agents := []AgentInfo{}
	r.logger.Debug("Store agent listing not yet implemented")
	return agents, nil
}

// PullAgent pulls an agent image from registry to local store
func (r *Resolver) PullAgent(registryRef string) error {
	r.logger.Info("Pulling agent from registry", "reference", registryRef)

	// Use existing remote pull functionality
	digest, err := remote.Pull(registryRef)
	if err != nil {
		return fmt.Errorf("pulling agent image: %w", err)
	}

	r.logger.Info("Successfully pulled agent", "reference", registryRef, "digest", digest)
	return nil
}

// fileExists checks if a file exists
func (r *Resolver) fileExists(path string) bool {
	_, err := os.Stat(r.expandPath(path))
	return err == nil
}

// expandPath expands ~ to home directory
func (r *Resolver) expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

// fromStore loads agent YAML content from content store (same logic as run.go)
func (r *Resolver) fromStore(reference string) (string, error) {
	img, err := r.store.GetArtifactImage(reference)
	if err != nil {
		return "", fmt.Errorf("getting image from store: %w", err)
	}

	layers, err := img.Layers()
	if err != nil {
		return "", fmt.Errorf("getting image layers: %w", err)
	}

	if len(layers) == 0 {
		return "", fmt.Errorf("image has no layers")
	}

	var buf bytes.Buffer
	layer := layers[0]
	b, err := layer.Uncompressed()
	if err != nil {
		return "", fmt.Errorf("uncompressing layer: %w", err)
	}
	defer b.Close()

	_, err = io.Copy(&buf, b)
	if err != nil {
		return "", fmt.Errorf("reading layer content: %w", err)
	}

	return buf.String(), nil
}
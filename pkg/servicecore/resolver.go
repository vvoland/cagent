// resolver.go implements secure agent specification resolution for cagent's multi-tenant architecture.
// This component handles the critical security boundary between external agent specifications
// and the cagent runtime system.
//
// Core Responsibilities:
// 1. Agent specification resolution with multiple source priority:
//   - File paths (absolute or relative to agents directory) - HIGHEST PRIORITY
//   - Content store lookups (Docker images) - FALLBACK
//   - Error reporting for missing agents - FINAL
//
// 2. Path security validation:
//   - Restricts all file access to within a configured root directory
//   - Prevents directory traversal attacks (../../../etc/passwd)
//   - Blocks access to system files outside the allowed scope
//   - Uses absolute path conversion with secure prefix matching
//
// 3. Agent discovery and metadata:
//   - Lists available file-based agents with recursive directory scanning
//   - Provides agent metadata for client consumption
//   - Integrates with Docker registry pulling for remote agents
//
// Security Architecture:
// The resolver implements defense-in-depth by:
// - Converting all paths to absolute form to eliminate relative path ambiguity
// - Using trailing separator prefix matching to prevent bypass attacks
// - Logging security violations for monitoring and audit
// - Failing secure by default (reject unsafe paths, don't expand home directories)
//
// This component is essential for preventing MCP clients from accessing arbitrary
// files on the host system while maintaining usability for legitimate agent specifications.
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
	rootDir   string // Security: restrict file access to this root directory
	store     *content.Store
}

// NewResolver creates a new agent resolver with security root directory
func NewResolver(agentsDir string) (*Resolver, error) {
	store, err := content.NewStore()
	if err != nil {
		return nil, fmt.Errorf("creating content store: %w", err)
	}

	return NewResolverWithStore(agentsDir, store)
}

// NewResolverWithStore creates a new agent resolver with a custom store (for testing)
func NewResolverWithStore(agentsDir string, store *content.Store) (*Resolver, error) {
	// Convert agentsDir to absolute path for security validation
	absAgentsDir, err := filepath.Abs(agentsDir)
	if err != nil {
		return nil, fmt.Errorf("converting agents directory to absolute path: %w", err)
	}

	return &Resolver{
		agentsDir: absAgentsDir,
		rootDir:   absAgentsDir, // Default root is the agents directory
		store:     store,
	}, nil
}

// isPathSafe validates that a path is within the allowed root directory
func (r *Resolver) isPathSafe(path string) error {
	// Convert target path to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("converting path to absolute: %w", err)
	}

	// Convert root directory to absolute path
	absRoot, err := filepath.Abs(r.rootDir)
	if err != nil {
		return fmt.Errorf("converting root directory to absolute: %w", err)
	}

	// Ensure root path ends with separator for proper prefix matching
	if !strings.HasSuffix(absRoot, string(filepath.Separator)) {
		absRoot += string(filepath.Separator)
	}

	// Check if target path starts with the root path prefix
	if !strings.HasPrefix(absPath+string(filepath.Separator), absRoot) {
		return fmt.Errorf("path outside allowed root directory")
	}

	return nil
}

// ResolveAgent resolves an agent specification to a file path with security restrictions
// Priority: File path (within root) → Content store → Error
func (r *Resolver) ResolveAgent(agentSpec string) (string, error) {
	slog.Debug("Resolving agent", "agent_spec", agentSpec)

	// First, try to resolve as file path (absolute or relative to agents dir)
	var candidatePath string
	if filepath.IsAbs(agentSpec) {
		candidatePath = agentSpec
	} else {
		candidatePath = filepath.Join(r.agentsDir, agentSpec)
	}

	// Security check: validate path is within allowed root
	if err := r.isPathSafe(candidatePath); err != nil {
		slog.Warn("Agent path rejected for security", "path", candidatePath, "error", err)
	} else if r.fileExists(candidatePath) {
		slog.Debug("Agent resolved as file path", "path", candidatePath)
		return candidatePath, nil
	}

	// If not a valid file path, try content store (Docker images)
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

	slog.Debug("Agent resolved from store", "agent_spec", agentSpec, "temp_file", tmpFile.Name())
	return tmpFile.Name(), nil
}

// ListFileAgents lists agents available as files
func (r *Resolver) ListFileAgents() ([]AgentInfo, error) {
	var agents []AgentInfo

	// Check if agents directory exists
	if !r.fileExists(r.agentsDir) {
		slog.Debug("Agents directory does not exist", "path", r.agentsDir)
		return agents, nil
	}

	// Walk through agents directory
	err := filepath.Walk(r.agentsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only include .yaml and .yml files
		if !info.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			relPath, err := filepath.Rel(r.agentsDir, path)
			if err != nil {
				relPath = path
			}

			agent := AgentInfo{
				Name:         strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
				Description:  fmt.Sprintf("File-based agent: %s", relPath),
				Source:       "file",
				Path:         path,    // Absolute path for internal resolution
				RelativePath: relPath, // Relative path for user reference
			}
			agents = append(agents, agent)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("listing file agents: %w", err)
	}

	slog.Debug("Listed file agents", "count", len(agents))
	return agents, nil
}

// ListStoreAgents lists agents available in the content store
func (r *Resolver) ListStoreAgents() ([]AgentInfo, error) {
	artifacts, err := r.store.ListArtifacts()
	if err != nil {
		return nil, fmt.Errorf("listing store artifacts: %w", err)
	}

	agents := make([]AgentInfo, 0, len(artifacts))
	for _, artifact := range artifacts {
		agent := AgentInfo{
			Name:        r.extractNameFromReference(artifact.Reference),
			Description: fmt.Sprintf("Store-based agent: %s", artifact.Reference),
			Source:      "store",
			Reference:   artifact.Reference, // Full image reference with tag (the agent ref)
		}
		agents = append(agents, agent)
	}

	slog.Debug("Listed store agents", "count", len(agents))
	return agents, nil
}

// extractNameFromReference extracts a friendly name from a reference
func (r *Resolver) extractNameFromReference(reference string) string {
	// Handle references like "docker.io/user/agent:tag" or "user/agent:tag"
	parts := strings.Split(reference, "/")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		// Remove tag if present
		if colonIndex := strings.LastIndex(lastPart, ":"); colonIndex != -1 {
			return lastPart[:colonIndex]
		}
		return lastPart
	}
	return reference
}

// PullAgent pulls an agent image from registry to local store
func (r *Resolver) PullAgent(registryRef string) error {
	slog.Info("Pulling agent from registry", "reference", registryRef)

	// Use existing remote pull functionality
	digest, err := remote.Pull(registryRef)
	if err != nil {
		return fmt.Errorf("pulling agent image: %w", err)
	}

	slog.Info("Successfully pulled agent", "reference", registryRef, "digest", digest)
	return nil
}

// fileExists checks if a file exists
func (r *Resolver) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
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

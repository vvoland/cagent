package teamloader

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/docker/cagent/pkg/content"
	"github.com/docker/cagent/pkg/remote"
)

type AgentSource interface {
	Name() string
	ParentDir() string
	Read(ctx context.Context) ([]byte, error)
}

type AgentSources map[string]AgentSource

// fileSource is used to load an agent configuration from a YAML file.
type fileSource struct {
	path string
}

func NewFileSource(path string) AgentSource {
	return fileSource{
		path: path,
	}
}

func (a fileSource) Name() string {
	return a.path
}

func (a fileSource) ParentDir() string {
	return filepath.Dir(a.path)
}

func (a fileSource) Read(context.Context) ([]byte, error) {
	parentDir := a.ParentDir()
	fs, err := os.OpenRoot(parentDir)
	if err != nil {
		return nil, fmt.Errorf("opening filesystem %s: %w", parentDir, err)
	}

	fileName := filepath.Base(a.path)
	data, err := fs.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", fileName, err)
	}

	return data, nil
}

// bytesSource is used to load an agent configuration from a []byte.
type bytesSource struct {
	data []byte
}

func NewBytesSource(data []byte) AgentSource {
	return bytesSource{
		data: data,
	}
}

func (a bytesSource) Name() string {
	return ""
}

func (a bytesSource) ParentDir() string {
	return ""
}

func (a bytesSource) Read(context.Context) ([]byte, error) {
	return a.data, nil
}

type ociSource struct {
	reference string
}

func NewOCISource(ref string) AgentSource {
	return ociSource{
		reference: ref,
	}
}

func (a ociSource) Name() string {
	return a.reference
}

func (a ociSource) ParentDir() string {
	return ""
}

func (a ociSource) Read(ctx context.Context) ([]byte, error) {
	// Check if we have a local copy (without loading content)
	store, err := content.NewStore()
	if err != nil {
		return nil, fmt.Errorf("failed to create content store: %w", err)
	}

	_, metaErr := store.GetArtifactMetadata(a.reference)
	hasLocal := metaErr == nil

	// Try to pull from remote (only pulls if digest changed)
	if _, pullErr := remote.Pull(ctx, a.reference, false); pullErr != nil {
		if !hasLocal {
			// No local copy and can't pull, error out
			return nil, fmt.Errorf("failed to pull OCI image %s: %w", a.reference, pullErr)
		}
		slog.Debug("Failed to check for OCI reference updates, using cached version", "ref", a.reference, "error", pullErr)
	}

	// Load the agent contents from the store
	af, err := store.GetArtifact(a.reference)
	if err != nil {
		return nil, fmt.Errorf("failed to load agent from store: %w", err)
	}

	return []byte(af), nil
}

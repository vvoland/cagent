package oci

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/static"
	"github.com/google/go-containerregistry/pkg/v1/types"

	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/config/latest"
	"github.com/docker/cagent/pkg/content"
	"github.com/docker/cagent/pkg/path"
	"github.com/docker/cagent/pkg/version"
)

// PackageFileAsOCIToStore creates an OCI artifact from a file and stores it in the content store
func PackageFileAsOCIToStore(ctx context.Context, filePath, artifactRef string, store *content.Store) (string, error) {
	if !strings.Contains(artifactRef, ":") {
		artifactRef += ":latest"
	}

	// Validate the file path to prevent directory traversal attacks
	validatedPath, err := path.ValidatePathInDirectory(filePath, "")
	if err != nil {
		return "", fmt.Errorf("invalid file path: %w", err)
	}

	data, err := os.ReadFile(validatedPath)
	if err != nil {
		return "", fmt.Errorf("reading file: %w", err)
	}

	cfg, err := config.LoadConfigBytes(ctx, data)
	if err != nil {
		return "", fmt.Errorf("loading config: %w", err)
	}

	var raw struct {
		Version string `yaml:"version,omitempty"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return "", fmt.Errorf("looking for version in config file\n%s", yaml.FormatError(err, true, true))
	}
	// Make sure we push a yaml with a version (Use latest if missing)
	if raw.Version == "" {
		cfg.Version = latest.Version
		data, err = yaml.MarshalWithOptions(cfg, yaml.Indent(2))
		if err != nil {
			return "", fmt.Errorf("marshaling config: %w", err)
		}
	}

	// Prepare OCI annotations
	annotations := map[string]string{
		"io.docker.cagent.version":             version.Version,
		"org.opencontainers.image.created":     time.Now().Format(time.RFC3339),
		"org.opencontainers.image.description": fmt.Sprintf("OCI artifact containing %s", filepath.Base(validatedPath)),
	}
	if author := cfg.Metadata.Author; author != "" {
		annotations["org.opencontainers.image.authors"] = author
	}
	if license := cfg.Metadata.License; license != "" {
		annotations["org.opencontainers.image.licenses"] = license
	}

	layer := static.NewLayer(data, types.OCIUncompressedLayer)
	img, err := mutate.AppendLayers(empty.Image, layer)
	if err != nil {
		return "", fmt.Errorf("appending layer: %w", err)
	}

	// Convert to OCI manifest format to support annotations
	img = mutate.MediaType(img, types.OCIManifestSchema1)
	img = mutate.Annotations(img, annotations).(v1.Image)

	digest, err := store.StoreArtifact(img, artifactRef)
	if err != nil {
		return "", fmt.Errorf("storing artifact in content store: %w", err)
	}

	return digest, nil
}

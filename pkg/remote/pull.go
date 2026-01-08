package remote

import (
	"context"
	"fmt"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"

	"github.com/docker/cagent/pkg/content"
)

// Pull pulls an artifact from a registry and stores it in the content store
func Pull(ctx context.Context, registryRef string, force bool, opts ...crane.Option) (string, error) {
	opts = append(opts, crane.WithContext(ctx))

	ref, err := name.ParseReference(registryRef)
	if err != nil {
		return "", fmt.Errorf("parsing registry reference %s: %w", registryRef, err)
	}

	// Use the full reference string to preserve registry information
	fullRef := ref.String()

	remoteDigest, err := crane.Digest(fullRef, opts...)
	if err != nil {
		return "", fmt.Errorf("resolving remote digest for %s: %w", registryRef, err)
	}

	store, err := content.NewStore()
	if err != nil {
		return "", fmt.Errorf("creating content store: %w", err)
	}

	if !force {
		if meta, metaErr := store.GetArtifactMetadata(fullRef); metaErr == nil {
			if meta.Digest == remoteDigest {
				if !hasCagentAnnotation(meta.Annotations) {
					return "", fmt.Errorf("artifact %s found in store wasn't created by `cagent push`\nTry to push again with `cagent push` (cagent >= v1.10.0)", fullRef)
				}
				return meta.Digest, nil
			}
		}
	}

	img, err := crane.Pull(fullRef, opts...)
	if err != nil {
		return "", fmt.Errorf("pulling image from registry %s: %w", registryRef, err)
	}

	manifest, err := img.Manifest()
	if err != nil {
		return "", fmt.Errorf("getting manifest from pulled image: %w", err)
	}
	if !hasCagentAnnotation(manifest.Annotations) {
		return "", fmt.Errorf("artifact %s wasn't created by `cagent push`\nTry to push again with `cagent push` (cagent >= v1.10.0)", fullRef)
	}

	digest, err := store.StoreArtifact(img, fullRef)
	if err != nil {
		return "", fmt.Errorf("storing artifact in content store: %w", err)
	}

	return digest, nil
}

func hasCagentAnnotation(annotations map[string]string) bool {
	_, exists := annotations["io.docker.cagent.version"]
	return exists
}

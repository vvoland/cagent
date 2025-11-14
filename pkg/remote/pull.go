package remote

import (
	"context"
	"fmt"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"

	"github.com/docker/cagent/pkg/content"
)

// Pull pulls an artifact from a registry and stores it in the content store
func Pull(ctx context.Context, registryRef string, opts ...crane.Option) (string, error) {
	opts = append(opts, crane.WithContext(ctx))

	ref, err := name.ParseReference(registryRef)
	if err != nil {
		return "", fmt.Errorf("parsing registry reference %s: %w", registryRef, err)
	}

	remoteDigest, err := crane.Digest(ref.String(), opts...)
	if err != nil {
		return "", fmt.Errorf("resolving remote digest for %s: %w", registryRef, err)
	}

	store, err := content.NewStore()
	if err != nil {
		return "", fmt.Errorf("creating content store: %w", err)
	}

	localRef := ref.Context().RepositoryStr() + ":" + ref.Identifier()
	if meta, metaErr := store.GetArtifactMetadata(localRef); metaErr == nil {
		if meta.Digest == remoteDigest {
			return meta.Digest, nil
		}
	}

	img, err := crane.Pull(ref.String(), opts...)
	if err != nil {
		return "", fmt.Errorf("pulling image from registry %s: %w", registryRef, err)
	}

	digest, err := store.StoreArtifact(img, localRef)
	if err != nil {
		return "", fmt.Errorf("storing artifact in content store: %w", err)
	}

	return digest, nil
}

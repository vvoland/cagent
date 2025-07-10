package remote

import (
	"testing"

	"github.com/docker/cagent/pkg/content"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/static"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

func TestPullNonExistentRegistry(t *testing.T) {
	_, err := Pull("registry.example.com/non-existent:latest")
	if err == nil {
		t.Fatal("Expected error for non-existent registry")
	}
}

func TestPullWithOptions(t *testing.T) {
	_, err := Pull("registry.example.com/test:latest", crane.Insecure)
	if err == nil {
		t.Fatal("Expected error for non-existent registry")
	}
}

func TestPullIntegration(t *testing.T) {
	store, err := content.NewStore(content.WithBaseDir(t.TempDir()))
	if err != nil {
		t.Fatalf("Failed to create content store: %v", err)
	}

	testData := []byte("test pull integration")
	layer := static.NewLayer(testData, types.OCIUncompressedLayer)
	img := empty.Image
	img, err = mutate.AppendLayers(img, layer)
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	testRef := "pull-test:latest"
	digest, err := store.StoreArtifact(img, testRef)
	if err != nil {
		t.Fatalf("Failed to store artifact: %v", err)
	}

	t.Cleanup(func() {
		if err := store.DeleteArtifact(digest); err != nil {
			t.Logf("Failed to clean up artifact: %v", err)
		}
	})

	retrievedImg, err := store.GetArtifactImage(testRef)
	if err != nil {
		t.Fatalf("Failed to retrieve artifact: %v", err)
	}

	if retrievedImg == nil {
		t.Fatal("Retrieved image is nil")
	}

	err = Push("invalid:reference:with:too:many:colons")
	if err == nil {
		t.Fatal("Expected error for invalid registry reference")
	}
}

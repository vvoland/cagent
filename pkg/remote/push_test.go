package remote

import (
	"testing"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/static"
	"github.com/google/go-containerregistry/pkg/v1/types"

	"github.com/docker/cagent/pkg/content"
)

func TestPush(t *testing.T) {
	store, err := content.NewStore(content.WithBaseDir(t.TempDir()))
	if err != nil {
		t.Fatalf("Failed to create content store: %v", err)
	}

	testData := []byte("test artifact data")

	layer := static.NewLayer(testData, types.OCIUncompressedLayer)
	img := empty.Image
	img, err = mutate.AppendLayers(img, layer)
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	testRef := "test-app:latest"
	digest, err := store.StoreArtifact(img, testRef)
	if err != nil {
		t.Fatalf("Failed to store artifact: %v", err)
	}

	t.Cleanup(func() {
		if err := store.DeleteArtifact(digest); err != nil {
			t.Logf("Failed to clean up artifact: %v", err)
		}
	})

	loadedImg, err := store.GetArtifactImage(testRef)
	if err != nil {
		t.Fatalf("Failed to load artifact by reference: %v", err)
	}

	if loadedImg == nil {
		t.Fatal("Loaded image is nil")
	}

	err = Push("invalid:reference:with:too:many:colons")
	if err == nil {
		t.Fatal("Expected error for invalid registry reference")
	}

	err = Push("invalid:reference:with:too:many:colons")
	if err == nil {
		t.Fatal("Expected error for invalid registry reference")
	}
}

func TestPushNonExistentArtifact(t *testing.T) {
	err := Push("registry.example.com/test:latest")
	if err == nil {
		t.Fatal("Expected error for non-existent artifact")
	}

	err = Push("registry.example.com/test:latest")
	if err == nil {
		t.Fatal("Expected error for non-existent artifact")
	}
}

func TestPushWithOptions(t *testing.T) {
	store, err := content.NewStore(content.WithBaseDir(t.TempDir()))
	if err != nil {
		t.Fatalf("Failed to create content store: %v", err)
	}

	testData := []byte("test artifact data with options")

	layer := static.NewLayer(testData, types.OCIUncompressedLayer)
	img := empty.Image
	img, err = mutate.AppendLayers(img, layer)
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	testRef := "test-app-options:v1.0.0"
	digest, err := store.StoreArtifact(img, testRef)
	if err != nil {
		t.Fatalf("Failed to store artifact: %v", err)
	}

	defer func() {
		if err := store.DeleteArtifact(digest); err != nil {
			t.Logf("Failed to clean up artifact: %v", err)
		}
	}()

	// Test with insecure option (this won't actually push anywhere)
	err = Push("invalid:reference:with:too:many:colons", crane.Insecure)
	if err == nil {
		t.Fatal("Expected error for invalid registry reference")
	}
}

func TestContentStore(t *testing.T) {
	// Create a content store
	store, err := content.NewStore(content.WithBaseDir(t.TempDir()))
	if err != nil {
		t.Fatalf("Failed to create content store: %v", err)
	}

	testData := []byte("test content store")

	layer := static.NewLayer(testData, types.OCIUncompressedLayer)
	img := empty.Image
	img, err = mutate.AppendLayers(img, layer)
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	testRef := "test-store:latest"
	digest, err := store.StoreArtifact(img, testRef)
	if err != nil {
		t.Fatalf("Failed to store artifact: %v", err)
	}

	t.Cleanup(func() {
		if err := store.DeleteArtifact(digest); err != nil {
			t.Logf("Failed to clean up artifact: %v", err)
		}
	})

	metadata, err := store.GetArtifactMetadata(testRef)
	if err != nil {
		t.Fatalf("Failed to get metadata: %v", err)
	}

	if metadata.Reference != testRef {
		t.Errorf("Expected reference %s, got %s", testRef, metadata.Reference)
	}

	if metadata.Digest != digest {
		t.Errorf("Expected digest %s, got %s", digest, metadata.Digest)
	}

	artifacts, err := store.ListArtifacts()
	if err != nil {
		t.Fatalf("Failed to list artifacts: %v", err)
	}

	found := false
	for _, artifact := range artifacts {
		if artifact.Reference == testRef {
			found = true
			break
		}
	}

	if !found {
		t.Error("Artifact not found in list")
	}
}

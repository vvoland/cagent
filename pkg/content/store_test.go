package content

import (
	"fmt"
	"testing"

	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/static"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

func TestStoreBasicOperations(t *testing.T) {
	store, err := NewStore(WithBaseDir(t.TempDir()))
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	testData := []byte("Hello, World! This is a test artifact.")
	layer := static.NewLayer(testData, types.OCIUncompressedLayer)
	img := empty.Image
	img, err = mutate.AppendLayers(img, layer)
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	testRef := "hello-world:v1.0.0"
	digest, err := store.StoreArtifact(img, testRef)
	if err != nil {
		t.Fatalf("Failed to store artifact: %v", err)
	}

	retrievedImg, err := store.GetArtifactImage(testRef)
	if err != nil {
		t.Fatalf("Failed to retrieve artifact by reference: %v", err)
	}

	if retrievedImg == nil {
		t.Fatal("Retrieved image is nil")
	}

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

func TestStoreMultipleArtifacts(t *testing.T) {
	store, err := NewStore(WithBaseDir(t.TempDir()))
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	testRefs := []string{
		"app1:v1.0.0",
		"app2:v2.0.0",
		"app3:latest",
	}

	var digests []string

	for i, ref := range testRefs {
		testData := []byte(fmt.Sprintf("Test artifact %d", i+1))
		layer := static.NewLayer(testData, types.OCIUncompressedLayer)
		img := empty.Image
		img, err = mutate.AppendLayers(img, layer)
		if err != nil {
			t.Fatalf("Failed to create test image %d: %v", i+1, err)
		}

		digest, err := store.StoreArtifact(img, ref)
		if err != nil {
			t.Fatalf("Failed to store artifact %d: %v", i+1, err)
		}

		digests = append(digests, digest)
	}

	artifacts, err := store.ListArtifacts()
	if err != nil {
		t.Fatalf("Failed to list artifacts: %v", err)
	}

	if len(artifacts) < len(testRefs) {
		t.Errorf("Expected at least %d artifacts, got %d", len(testRefs), len(artifacts))
	}

	for _, ref := range testRefs {
		img, err := store.GetArtifactImage(ref)
		if err != nil {
			t.Errorf("Failed to retrieve artifact %s: %v", ref, err)
		}
		if img == nil {
			t.Errorf("Retrieved image for %s is nil", ref)
		}
	}
}

func TestStoreResolution(t *testing.T) {
	store, err := NewStore(WithBaseDir(t.TempDir()))
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	testData := []byte("Resolution test artifact")
	layer := static.NewLayer(testData, types.OCIUncompressedLayer)
	img := empty.Image
	img, err = mutate.AppendLayers(img, layer)
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	testRef := "resolution-test:latest"
	_, err = store.StoreArtifact(img, testRef)
	if err != nil {
		t.Fatalf("Failed to store artifact: %v", err)
	}

	testCases := []string{
		testRef,
	}

	for _, tc := range testCases {
		img, err := store.GetArtifactImage(tc)
		if err != nil {
			t.Errorf("Failed to resolve identifier %s: %v", tc, err)
		}
		if img == nil {
			t.Errorf("Retrieved image for %s is nil", tc)
		}
	}
}

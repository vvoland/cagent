package remote

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/static"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/pkg/content"
)

func TestPullRegistryNotFound(t *testing.T) {
	t.Parallel()

	// Use a test server that returns 404 for fast failure
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Extract host:port from server URL (remove http://)
	registry := strings.TrimPrefix(server.URL, "http://")

	// Test various image references that should fail with 404
	refs := []string{
		registry + "/non-existent:latest",
		registry + "/test:latest",
	}

	for _, ref := range refs {
		_, err := Pull(t.Context(), ref, false, crane.Insecure)
		require.Error(t, err, "expected error for ref: %s", ref)
	}
}

func TestPullIntegration(t *testing.T) {
	t.Parallel()

	store, err := content.NewStore(content.WithBaseDir(t.TempDir()))
	require.NoError(t, err)

	testData := []byte("test pull integration")
	layer := static.NewLayer(testData, types.OCIUncompressedLayer)
	img := empty.Image
	img, err = mutate.AppendLayers(img, layer)
	require.NoError(t, err)

	testRef := "pull-test:latest"
	digest, err := store.StoreArtifact(img, testRef)
	require.NoError(t, err)

	t.Cleanup(func() {
		if err := store.DeleteArtifact(digest); err != nil {
			t.Logf("Failed to clean up artifact: %v", err)
		}
	})

	retrievedImg, err := store.GetArtifactImage(testRef)
	require.NoError(t, err)
	assert.NotNil(t, retrievedImg)

	err = Push("invalid:reference:with:too:many:colons")
	require.Error(t, err)
}

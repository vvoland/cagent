package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/pkg/config/latest"
)

// TestExampleYAMLRoundtrip tests that all example YAML files can be serialized/deserialized
// without losing information. This simulates the push/pull flow where configs are:
// 1. Read from YAML file
// 2. Parsed into Config struct
// 3. Marshaled back to YAML (for OCI packaging)
// 4. Unmarshaled again (when pulled)
func TestExampleYAMLRoundtrip(t *testing.T) {
	t.Parallel()

	examplesDir := filepath.Join("..", "..", "examples")
	if _, err := os.Stat(examplesDir); os.IsNotExist(err) {
		t.Skip("examples directory not found")
	}

	// Collect all YAML files from examples directory (including subdirectories)
	var yamlFiles []string
	err := filepath.Walk(examplesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (filepath.Ext(path) == ".yaml" || filepath.Ext(path) == ".yml") {
			yamlFiles = append(yamlFiles, path)
		}
		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, yamlFiles, "no YAML files found in examples directory")

	ctx := t.Context()

	for _, yamlFile := range yamlFiles {
		t.Run(filepath.Base(yamlFile), func(t *testing.T) {
			t.Parallel()

			// Step 1: Read original YAML file
			originalBytes, err := os.ReadFile(yamlFile)
			require.NoError(t, err, "failed to read %s", yamlFile)

			// Step 2: Parse into Config struct (simulates push reading the file)
			cfg, err := LoadConfigBytes(ctx, originalBytes)
			require.NoError(t, err, "failed to load config from %s", yamlFile)
			require.NotNil(t, cfg, "config should not be nil for %s", yamlFile)

			// Step 3: Marshal back to YAML with same options as push command
			// (see pkg/oci/package.go PackageFileAsOCIToStore)
			marshaledBytes, err := yaml.MarshalWithOptions(cfg, yaml.Indent(2))
			require.NoError(t, err, "failed to marshal config for %s", yamlFile)

			// Step 4: Unmarshal again (simulates pull reading from OCI)
			var cfg2 *latest.Config
			cfg2, err = LoadConfigBytes(ctx, marshaledBytes)
			require.NoError(t, err, "failed to load marshaled config for %s", yamlFile)
			require.NotNil(t, cfg2, "round-tripped config should not be nil for %s", yamlFile)

			// Step 5: Compare the two parsed configs - they should be identical
			// Marshal both to JSON for easy comparison (avoids YAML formatting differences)
			assertConfigsEqual(t, cfg, cfg2, yamlFile)

			// Step 6: Ensure the marshaled YAML can be marshaled again identically (stability test)
			marshaledBytes2, err := yaml.MarshalWithOptions(cfg2, yaml.Indent(2))
			require.NoError(t, err, "failed to re-marshal config for %s", yamlFile)

			// Parse both marshaled versions and compare
			var cfg3 *latest.Config
			cfg3, err = LoadConfigBytes(ctx, marshaledBytes2)
			require.NoError(t, err, "failed to load re-marshaled config for %s", yamlFile)

			assertConfigsEqual(t, cfg, cfg3, yamlFile)
		})
	}
}

// assertConfigsEqual compares two configs for semantic equality using go-cmp
func assertConfigsEqual(t *testing.T, cfg1, cfg2 *latest.Config, filename string) {
	t.Helper()

	// Define comparison options to handle normalization and special cases
	opts := []cmp.Option{
		// Sort maps for consistent comparison (map iteration order is random)
		cmpopts.SortMaps(func(a, b string) bool { return a < b }),

		// Handle ParallelToolCalls normalization: nil and &true are considered equal
		// Config validation sets ParallelToolCalls to &true if nil, so after roundtrip it may differ
		cmp.Comparer(func(a, b *bool) bool {
			// Both nil is equal
			if a == nil && b == nil {
				return true
			}
			// One nil, one true is equal (normalized case)
			if (a == nil && b != nil && *b == true) || (b == nil && a != nil && *a == true) {
				return true
			}
			// Both non-nil, compare values
			if a != nil && b != nil {
				return *a == *b
			}
			return false
		}),

		// Handle RAGDatabaseConfig which has unexported fields
		// Compare using the public AsString() method
		cmp.Comparer(func(a, b latest.RAGDatabaseConfig) bool {
			aStr, aErr := a.AsString()
			bStr, bErr := b.AsString()
			// If both error, consider them equal (both invalid)
			if aErr != nil && bErr != nil {
				return true
			}
			// If one errors, not equal
			if aErr != nil || bErr != nil {
				return false
			}
			// Compare the string values
			return aStr == bStr
		}),

		// Treat nil and empty slices as equal (common normalization during YAML marshal/unmarshal)
		cmpopts.EquateEmpty(),
	}

	// Use cmp.Diff to get detailed differences
	if diff := cmp.Diff(cfg1, cfg2, opts...); diff != "" {
		t.Errorf("Config mismatch for %s (-want +got):\n%s", filename, diff)
	}
}

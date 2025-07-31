package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrate_v0_v1_provider(t *testing.T) {
	cfg, err := LoadConfig("testdata/pirate_v0.yaml")
	require.NoError(t, err)

	assert.Equal(t, "openai", cfg.Models["openai"].Provider)
}

func TestMigrate_v1_provider(t *testing.T) {
	cfg, err := LoadConfig("testdata/pirate_v1.yaml")
	require.NoError(t, err)

	assert.Equal(t, "openai", cfg.Models["openai"].Provider)
}

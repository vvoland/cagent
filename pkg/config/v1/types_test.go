package v1

import (
	"os"
	"testing"

	v0 "github.com/docker/cagent/pkg/config/v0"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestMigrate_v0_v1_provider(t *testing.T) {
	data, err := os.ReadFile("testdata/pirate_v0.yaml")
	require.NoError(t, err)

	var oldConfig v0.Config
	err = yaml.Unmarshal(data, &oldConfig)
	require.NoError(t, err)

	require.NoError(t, err)
	assert.Equal(t, "openai", oldConfig.Models["openai"].Type)

	newConfig := UpgradeFrom(oldConfig)

	assert.Equal(t, "openai", newConfig.Models["openai"].Provider)
}

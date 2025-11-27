package config

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestV2Commands_AllForms(t *testing.T) {
	cfg, err := Load(t.Context(), fileSource("testdata/commands_v2.yaml"))
	require.NoError(t, err)

	cmdsMap := cfg.Agents["root"].Commands
	require.Equal(t, "check disk", cmdsMap["df"])
	require.Equal(t, "list files", cmdsMap["ls"])

	cmdsList := cfg.Agents["another_agent"].Commands
	require.Equal(t, "check disk", cmdsList["df"])
	require.Equal(t, "list files", cmdsList["ls"])

	require.Empty(t, cfg.Agents["none_agent"].Commands)
}

func TestMigrate_v1_Commands_AllForms(t *testing.T) {
	cfg, err := Load(t.Context(), fileSource("testdata/commands_v1.yaml"))
	require.NoError(t, err)

	cmdsMap := cfg.Agents["root"].Commands
	require.Equal(t, "check disk", cmdsMap["df"])
	require.Equal(t, "list files", cmdsMap["ls"])

	cmdsList := cfg.Agents["another_agent"].Commands
	require.Equal(t, "check disk", cmdsList["df"])
	require.Equal(t, "list files", cmdsList["ls"])

	require.Empty(t, cfg.Agents["yet_another_agent"].Commands)
}

func TestMigrate_v0_Commands_AllForms(t *testing.T) {
	cfg, err := Load(t.Context(), fileSource("testdata/commands_v0.yaml"))
	require.NoError(t, err)

	cmdsMap := cfg.Agents["root"].Commands
	require.Equal(t, "check disk", cmdsMap["df"])
	require.Equal(t, "list files", cmdsMap["ls"])

	cmdsList := cfg.Agents["another_agent"].Commands
	require.Equal(t, "check disk", cmdsList["df"])
	require.Equal(t, "list files", cmdsList["ls"])

	require.Empty(t, cfg.Agents["yet_another_agent"].Commands)
}

type fileSource string

func (s fileSource) Read(context.Context) ([]byte, error) {
	return os.ReadFile(string(s))
}

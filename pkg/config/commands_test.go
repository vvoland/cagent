package config

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestV2Commands_AllForms(t *testing.T) {
	cfg, err := Load(t.Context(), testfileSource("testdata/commands_v2.yaml"))
	require.NoError(t, err)

	// Test simple map format
	cmdsMap := cfg.Agents["root"].Commands
	require.Equal(t, "check disk", cmdsMap["df"].Instruction)
	require.Equal(t, "list files", cmdsMap["ls"].Instruction)
	require.Empty(t, cmdsMap["df"].Description) // Simple format has no description

	// Test list format
	cmdsList := cfg.Agents["another_agent"].Commands
	require.Equal(t, "check disk", cmdsList["df"].Instruction)
	require.Equal(t, "list files", cmdsList["ls"].Instruction)

	// Test advanced format with description and instruction
	cmdsAdvanced := cfg.Agents["advanced_agent"].Commands
	require.Equal(t, "Fix linting errors in the codebase", cmdsAdvanced["fix-lint"].Description)
	require.Equal(t, "Fix the lint issues", cmdsAdvanced["fix-lint"].Instruction)
	require.Equal(t, "Analyze the code", cmdsAdvanced["analyze"].Description)
	require.Contains(t, cmdsAdvanced["analyze"].Instruction, "$1")
	require.Contains(t, cmdsAdvanced["analyze"].Instruction, "$2")

	// Test empty commands
	require.Empty(t, cfg.Agents["none_agent"].Commands)
}

func TestV2Commands_DisplayText(t *testing.T) {
	cfg, err := Load(t.Context(), testfileSource("testdata/commands_v2.yaml"))
	require.NoError(t, err)

	// Simple format: DisplayText returns the instruction
	simpleCmd := cfg.Agents["root"].Commands["df"]
	require.Equal(t, "check disk", simpleCmd.DisplayText())

	// Advanced format: DisplayText returns description if available
	advancedCmd := cfg.Agents["advanced_agent"].Commands["fix-lint"]
	require.Equal(t, "Fix linting errors in the codebase", advancedCmd.DisplayText())
}

func TestMigrate_v1_Commands_AllForms(t *testing.T) {
	cfg, err := Load(t.Context(), testfileSource("testdata/commands_v1.yaml"))
	require.NoError(t, err)

	cmdsMap := cfg.Agents["root"].Commands
	require.Equal(t, "check disk", cmdsMap["df"].Instruction)
	require.Equal(t, "list files", cmdsMap["ls"].Instruction)

	cmdsList := cfg.Agents["another_agent"].Commands
	require.Equal(t, "check disk", cmdsList["df"].Instruction)
	require.Equal(t, "list files", cmdsList["ls"].Instruction)

	require.Empty(t, cfg.Agents["yet_another_agent"].Commands)
}

func TestMigrate_v0_Commands_AllForms(t *testing.T) {
	cfg, err := Load(t.Context(), testfileSource("testdata/commands_v0.yaml"))
	require.NoError(t, err)

	cmdsMap := cfg.Agents["root"].Commands
	require.Equal(t, "check disk", cmdsMap["df"].Instruction)
	require.Equal(t, "list files", cmdsMap["ls"].Instruction)

	cmdsList := cfg.Agents["another_agent"].Commands
	require.Equal(t, "check disk", cmdsList["df"].Instruction)
	require.Equal(t, "list files", cmdsList["ls"].Instruction)

	require.Empty(t, cfg.Agents["yet_another_agent"].Commands)
}

type testfileSource string

func (s testfileSource) Read(context.Context) ([]byte, error) {
	return os.ReadFile(string(s))
}

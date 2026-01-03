package v2

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/pkg/config/types"
)

func TestCommandsUnmarshal_Map(t *testing.T) {
	var c types.Commands
	input := []byte(`
df: "check disk"
ls: "list files"
`)
	err := yaml.Unmarshal(input, &c)
	require.NoError(t, err)
	require.Equal(t, "check disk", c["df"].Instruction)
	require.Equal(t, "list files", c["ls"].Instruction)
}

func TestCommandsUnmarshal_List(t *testing.T) {
	var c types.Commands
	input := []byte(`
- df: "check disk"
- ls: "list files"
`)
	err := yaml.Unmarshal(input, &c)
	require.NoError(t, err)
	require.Equal(t, "check disk", c["df"].Instruction)
	require.Equal(t, "list files", c["ls"].Instruction)
}

func TestCommandsUnmarshal_Advanced(t *testing.T) {
	var c types.Commands
	input := []byte(`
fix-lint:
  description: "Fix linting errors"
  instruction: "Fix the lint issues"
simple: "A simple command"
`)
	err := yaml.Unmarshal(input, &c)
	require.NoError(t, err)
	require.Equal(t, "Fix linting errors", c["fix-lint"].Description)
	require.Equal(t, "Fix the lint issues", c["fix-lint"].Instruction)
	require.Equal(t, "A simple command", c["simple"].Instruction)
	require.Empty(t, c["simple"].Description)
}

func TestCommandsDisplayText(t *testing.T) {
	simple := types.Command{Instruction: "simple instruction"}
	require.Equal(t, "simple instruction", simple.DisplayText())

	advanced := types.Command{Description: "my description", Instruction: "instruction"}
	require.Equal(t, "my description", advanced.DisplayText())
}

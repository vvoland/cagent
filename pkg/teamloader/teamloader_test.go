package teamloader

import (
	"context"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/pkg/agentfile"
	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/config/latest"
	"github.com/docker/cagent/pkg/environment"
)

func collectExamples(t *testing.T) []string {
	t.Helper()

	var files []string
	err := filepath.WalkDir(filepath.Join("..", "..", "examples"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".yaml" {
			files = append(files, path)
		}
		return nil
	})
	require.NoError(t, err)
	assert.NotEmpty(t, files)

	return files
}

type noEnvProvider struct{}

func (p *noEnvProvider) Get(context.Context, string) string { return "" }

func TestGetToolsForAgent_ContinuesOnCreateToolError(t *testing.T) {
	t.Parallel()

	// Agent with a bogus toolset type to force createTool error without network access
	a := &latest.AgentConfig{
		Instruction: "test",
		Toolsets:    []latest.Toolset{{Type: "does-not-exist"}},
	}

	runConfig := config.RuntimeConfig{
		EnvProviderForTests: &noEnvProvider{},
	}

	got, warnings := getToolsForAgent(t.Context(), a, ".", &runConfig, NewToolsetRegistry())

	require.Empty(t, got)
	require.NotEmpty(t, warnings)
	require.Contains(t, warnings[0], "toolset does-not-exist failed")
}

func TestLoadExamples(t *testing.T) {
	// Collect the missing env vars.
	missingEnvs := map[string]bool{}

	runtimeConfig := &config.RuntimeConfig{}

	for _, agentFilename := range collectExamples(t) {
		t.Run(agentFilename, func(t *testing.T) {
			source, err := agentfile.Resolve(agentFilename)
			require.NoError(t, err)

			_, err = Load(t.Context(), source, runtimeConfig)
			if err != nil {
				envErr := &environment.RequiredEnvError{}
				require.ErrorAs(t, err, &envErr)

				for _, env := range envErr.Missing {
					missingEnvs[env] = true
				}
			}
		})
	}

	for name := range missingEnvs {
		t.Setenv(name, "dummy")
	}

	// Load all the examples.
	for _, agentFilename := range collectExamples(t) {
		t.Run(agentFilename, func(t *testing.T) {
			t.Parallel()

			source, err := agentfile.Resolve(agentFilename)
			require.NoError(t, err)

			teams, err := Load(t.Context(), source, runtimeConfig)
			require.NoError(t, err)
			assert.NotEmpty(t, teams)
		})
	}
}

func TestLoadDefaultAgent(t *testing.T) {
	t.Parallel()

	source, err := agentfile.Resolve("../../pkg/agentfile/default-agent.yaml")
	require.NoError(t, err)

	teams, err := Load(t.Context(), source, &config.RuntimeConfig{})
	require.NoError(t, err)
	require.NotEmpty(t, teams)
}

func TestOverrideModel(t *testing.T) {
	tests := []struct {
		overrides   []string
		expected    string
		expectedErr string
	}{
		{
			overrides: []string{"anthropic/claude-4-6"},
			expected:  "anthropic/claude-4-6",
		},
		{
			overrides: []string{"root=anthropic/claude-4-6"},
			expected:  "anthropic/claude-4-6",
		},
		{
			overrides:   []string{"missing=anthropic/claude-4-6"},
			expectedErr: "unknown agent 'missing'",
		},
	}

	t.Setenv("OPENAI_API_KEY", "asdf")
	t.Setenv("ANTHROPIC_API_KEY", "asdf")

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			t.Parallel()

			source, err := agentfile.Resolve("testdata/basic.yaml")
			require.NoError(t, err)

			team, err := Load(t.Context(), source, &config.RuntimeConfig{}, WithModelOverrides(test.overrides))
			if test.expectedErr != "" {
				require.Contains(t, err.Error(), test.expectedErr)
			} else {
				require.NoError(t, err)
				rootAgent, err := team.Agent("root")
				require.NoError(t, err)
				require.Equal(t, test.expected, rootAgent.Model().ID())
			}
		})
	}
}

func TestToolsetInstructions(t *testing.T) {
	source, err := agentfile.Resolve("testdata/tool-instruction.yaml")
	require.NoError(t, err)

	team, err := Load(t.Context(), source, &config.RuntimeConfig{})
	require.NoError(t, err)

	agent, err := team.Agent("root")
	require.NoError(t, err)

	toolsets := agent.ToolSets()
	require.Len(t, toolsets, 1)

	instructions := toolsets[0].Instructions()
	expected := "Dummy fetch tool instruction"
	require.Equal(t, expected, instructions)
}

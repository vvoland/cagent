package root

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/teamloader"
)

func TestMCPCommand(t *testing.T) {
	t.Run("mcp_basic_agent_single_tool", func(t *testing.T) {
		ctx := t.Context()

		runConfig := config.RuntimeConfig{
			RedirectURI: "http://localhost:8083/oauth-callback",
		}
		team, err := teamloader.Load(ctx, "../../examples/basic_agent.yaml", runConfig)
		require.NoError(t, err, "Failed to load agent")
		defer func() {
			if err := team.StopToolSets(ctx); err != nil {
				t.Logf("Failed to stop tool sets: %v", err)
			}
		}()

		agentNames := team.AgentNames()
		require.Len(t, agentNames, 1)
		assert.Equal(t, "root", agentNames[0])

		agent, err := team.Agent("root")
		require.NoError(t, err)
		description := agent.Description()
		assert.Contains(t, description, "helpful AI assistant")

		handler := createToolHandler(team, "root", "../../examples/basic_agent.yaml")
		toolInput := ToolInput{Message: "What is 2+2? Answer in one sentence."}
		_, output, err := handler(ctx, nil, toolInput)

		require.NoError(t, err)
		assert.NotEmpty(t, output.Response)
	})

	t.Run("mcp_multi_agent_team", func(t *testing.T) {
		ctx := t.Context()

		runConfig := config.RuntimeConfig{
			RedirectURI: "http://localhost:8083/oauth-callback",
		}
		team, err := teamloader.Load(ctx, "../../examples/multi-code.yaml", runConfig)
		require.NoError(t, err, "Failed to load team")
		defer func() {
			if err := team.StopToolSets(ctx); err != nil {
				t.Logf("Failed to stop tool sets: %v", err)
			}
		}()

		agentNames := team.AgentNames()
		require.GreaterOrEqual(t, len(agentNames), 3)

		expectedAgents := []string{"root", "web", "golang"}
		for _, expectedName := range expectedAgents {
			assert.Contains(t, agentNames, expectedName)
		}

		handler := createToolHandler(team, "web", "../../examples/multi-code.yaml")
		toolInput := ToolInput{Message: "Say hello in one sentence."}
		_, output, err := handler(ctx, nil, toolInput)

		require.NoError(t, err)
		assert.NotEmpty(t, output.Response)
	})
}

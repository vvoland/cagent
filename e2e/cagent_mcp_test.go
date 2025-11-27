package e2e_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/pkg/agentfile"
	"github.com/docker/cagent/pkg/mcp"
	"github.com/docker/cagent/pkg/teamloader"
)

func TestMCP_SingleAgent(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	agentSource, err := agentfile.Resolve("testdata/basic.yaml")
	require.NoError(t, err)

	_, runConfig := startRecordingAIProxy(t)
	team, err := teamloader.Load(ctx, agentSource, runConfig)
	require.NoError(t, err, "Failed to load agent")
	t.Cleanup(func() {
		require.NoError(t, team.StopToolSets(ctx))
	})

	handler := mcp.CreateToolHandler(team, "root")
	_, output, err := handler(ctx, nil, mcp.ToolInput{
		Message: "What is 2+2? Answer in one sentence.",
	})

	require.NoError(t, err)
	assert.Equal(t, "2 + 2 equals 4.", output.Response)
}

func TestMCP_MultiAgent(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	agentSource, err := agentfile.Resolve("testdata/multi.yaml")
	require.NoError(t, err)

	_, runConfig := startRecordingAIProxy(t)
	team, err := teamloader.Load(ctx, agentSource, runConfig)
	require.NoError(t, err, "Failed to load team")
	t.Cleanup(func() {
		require.NoError(t, team.StopToolSets(ctx))
	})

	handler := mcp.CreateToolHandler(team, "web")
	_, output, err := handler(ctx, nil, mcp.ToolInput{
		Message: "Say hello in one sentence.",
	})

	require.NoError(t, err)
	assert.Equal(t, "Hello — how can I help you today?", output.Response)
}

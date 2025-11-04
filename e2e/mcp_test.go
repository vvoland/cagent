package e2e_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/cmd/root"
	"github.com/docker/cagent/pkg/teamloader"
)

func TestMCPSingleAgent(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	_, runtimeConfig := startRecordingAIProxy(t)

	team, err := teamloader.Load(ctx, "testdata/basic.yaml", runtimeConfig)
	require.NoError(t, err, "Failed to load agent")
	t.Cleanup(func() {
		require.NoError(t, team.StopToolSets(ctx))
	})

	handler := root.CreateToolHandler(team, "root", "testdata/basic.yaml")
	_, output, err := handler(ctx, nil, root.ToolInput{
		Message: "What is 2+2? Answer in one sentence.",
	})

	require.NoError(t, err)
	assert.Equal(t, "2+2 equals 4.", output.Response)
}

func TestMCPMultiAgent(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	_, runtimeConfig := startRecordingAIProxy(t)

	team, err := teamloader.Load(ctx, "testdata/multi.yaml", runtimeConfig)
	require.NoError(t, err, "Failed to load team")
	t.Cleanup(func() {
		require.NoError(t, team.StopToolSets(ctx))
	})

	handler := root.CreateToolHandler(team, "web", "testdata/multi.yaml")
	_, output, err := handler(ctx, nil, root.ToolInput{
		Message: "Say hello in one sentence.",
	})

	require.NoError(t, err)
	assert.Equal(t, "Hello!", output.Response)
}

package e2e_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/pkg/agentfile"
	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/teamloader"
)

func TestRuntime_OpenAI_Basic(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	agentSource, err := agentfile.Resolve("testdata/basic.yaml")
	require.NoError(t, err)

	_, runtimeConfig := startRecordingAIProxy(t)
	team, err := teamloader.Load(ctx, agentSource, runtimeConfig)
	require.NoError(t, err)

	rt, err := runtime.New(team)
	require.NoError(t, err)

	sess := session.New(session.WithUserMessage("What's 2+2?"))
	_, err = rt.Run(ctx, sess)
	require.NoError(t, err)

	response := sess.GetLastAssistantMessageContent()
	assert.Equal(t, "2 + 2 is equal to 4.", response)
	assert.Equal(t, "Simple Math Calculation", sess.Title)
}

func TestRuntime_Mistral_Basic(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	agentSource, err := agentfile.Resolve("testdata/basic.yaml")
	require.NoError(t, err)

	_, runtimeConfig := startRecordingAIProxy(t)
	team, err := teamloader.Load(ctx, agentSource, runtimeConfig, teamloader.WithModelOverrides([]string{"mistral/mistral-small"}))
	require.NoError(t, err)

	rt, err := runtime.New(team)
	require.NoError(t, err)

	sess := session.New(session.WithUserMessage("What's 2+2?"))
	_, err = rt.Run(ctx, sess)
	require.NoError(t, err)

	response := sess.GetLastAssistantMessageContent()
	assert.Equal(t, "The sum of 2 + 2 is 4.", response)
	assert.Equal(t, "Basic Arithmetic: Sum of 2 and 2", sess.Title)
}

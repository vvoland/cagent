package e2e_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/teamloader"
)

func TestRuntime_OpenAI_Basic(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	_, runtimeConfig := startRecordingAIProxy(t)

	team, err := teamloader.Load(ctx, "testdata/basic.yaml", runtimeConfig)
	require.NoError(t, err)

	rt, err := runtime.New(team)
	require.NoError(t, err)

	sess := session.New(session.WithUserMessage("", "What's 2+2?"))
	_, err = rt.Run(ctx, sess)
	require.NoError(t, err)

	response := sess.GetLastAssistantMessageContent()
	assert.Equal(t, "2 + 2 equals 4.", response)
	assert.Equal(t, "Basic Arithmetic: Addition of 2 and 2", sess.Title)
}

func TestRuntime_Mistral_Basic(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	_, runtimeConfig := startRecordingAIProxy(t)

	team, err := teamloader.Load(ctx, "testdata/basic.yaml", runtimeConfig, teamloader.WithModelOverrides([]string{"mistral/mistral-small"}))
	require.NoError(t, err)

	rt, err := runtime.New(team)
	require.NoError(t, err)

	sess := session.New(session.WithUserMessage("", "What's 2+2?"))
	_, err = rt.Run(ctx, sess)
	require.NoError(t, err)

	response := sess.GetLastAssistantMessageContent()
	assert.Equal(t, "The sum of 2 + 2 is 4.", response)
	assert.Equal(t, "Math Basics: Simple Addition", sess.Title)
}

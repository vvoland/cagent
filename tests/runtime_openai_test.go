package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/environment"
	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/teamloader"
)

func TestRuntime_BasicOpenAI(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	svr := startRecordingAIProxy(t)

	team, err := teamloader.Load(ctx, "testdata/basic.yaml", config.RuntimeConfig{
		ModelsGateway: svr.URL,
		DefaultEnvProvider: &testEnvProvider{
			environment.DockerDesktopTokenEnv: "DUMMY",
		},
	})
	require.NoError(t, err)

	rt, err := runtime.New(team)
	require.NoError(t, err)

	sess := session.New(session.WithUserMessage("", "Who's djordje?"))
	messages, err := rt.Run(ctx, sess)
	require.NoError(t, err)

	response := messages[len(messages)-1].Message.Content
	require.NoError(t, err)
	assert.Equal(t, "Djordje is a common Serbian name. It is used to address or refer to someone named Djordje. If you have more specific information or context related to a Djordje, please provide it so I can assist you better.", response)
}

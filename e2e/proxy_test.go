package e2e_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"

	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/environment"
	"github.com/docker/cagent/pkg/fake"
)

func startRecordingAIProxy(t *testing.T) (*httptest.Server, *config.RuntimeConfig) {
	t.Helper()

	cassettePath := filepath.Join("testdata", "cassettes", t.Name())

	// Create a matcher that fails the test on error
	matcher := fake.CustomMatcher(func(err error) {
		require.NoError(t, err)
	})

	// Header updater that adds real API keys for recording
	headerUpdater := func(host string, req *http.Request) {
		switch host {
		case "https://api.openai.com/v1":
			req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))
		case "https://api.anthropic.com":
			req.Header.Del("Authorization")
			req.Header.Set("X-Api-Key", os.Getenv("ANTHROPIC_API_KEY"))
		case "https://generativelanguage.googleapis.com":
			req.Header.Del("Authorization")
			req.Header.Set("X-Goog-Api-Key", os.Getenv("GOOGLE_API_KEY"))
		case "https://api.mistral.ai/v1":
			req.Header.Set("Authorization", "Bearer "+os.Getenv("MISTRAL_API_KEY"))
		}
	}

	proxyURL, cleanup, err := fake.StartProxyWithOptions(
		cassettePath,
		recorder.ModeRecordOnce,
		matcher,
		headerUpdater,
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, cleanup())
	})

	return &httptest.Server{URL: proxyURL}, &config.RuntimeConfig{
		Config: config.Config{
			ModelsGateway: proxyURL,
		},
		EnvProviderForTests: &testEnvProvider{
			environment.DockerDesktopTokenEnv: "DUMMY",
		},
	}
}

type testEnvProvider map[string]string

func (p *testEnvProvider) Get(_ context.Context, name string) string {
	return (*p)[name]
}

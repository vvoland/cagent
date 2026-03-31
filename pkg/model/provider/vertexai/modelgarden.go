// Package vertexai provides support for non-Gemini models hosted on
// Google Cloud's Vertex AI Model Garden via the OpenAI-compatible endpoint.
//
// Vertex AI Model Garden hosts models from various publishers (Anthropic,
// Meta, Mistral, etc.) and exposes them through an OpenAI-compatible API.
// This package configures the OpenAI provider to talk to that endpoint
// using Google Cloud Application Default Credentials for authentication.
//
// Usage in agent config:
//
//	models:
//	  claude-on-vertex:
//	    provider: google
//	    model: claude-sonnet-4-20250514
//	    provider_opts:
//	      project: my-gcp-project
//	      location: us-east5
//	      publisher: anthropic
package vertexai

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/docker/docker-agent/pkg/config/latest"
	"github.com/docker/docker-agent/pkg/environment"
	"github.com/docker/docker-agent/pkg/model/provider/openai"
	"github.com/docker/docker-agent/pkg/model/provider/options"
)

// cloudPlatformScope is the OAuth2 scope required for Vertex AI API access.
const cloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"

// validGCPIdentifier matches GCP project IDs and location names.
// Project IDs: 6-30 chars, lowercase letters, digits, hyphens.
// Locations: lowercase letters, digits, hyphens (e.g. us-central1).
var validGCPIdentifier = regexp.MustCompile(`^[a-z][a-z0-9-]{1,29}$`)

// IsModelGardenConfig returns true when the ModelConfig describes a
// non-Gemini model on Vertex AI (i.e. the "publisher" provider_opt is set).
func IsModelGardenConfig(cfg *latest.ModelConfig) bool {
	if cfg == nil || cfg.ProviderOpts == nil {
		return false
	}
	publisher, _ := cfg.ProviderOpts["publisher"].(string)
	return publisher != "" && !strings.EqualFold(publisher, "google")
}

// NewClient creates an OpenAI-compatible client pointing at the Vertex AI
// Model Garden endpoint. It uses Google Application Default Credentials
// for authentication.
func NewClient(ctx context.Context, cfg *latest.ModelConfig, env environment.Provider, opts ...options.Opt) (*openai.Client, error) {
	project, _ := cfg.ProviderOpts["project"].(string)
	location, _ := cfg.ProviderOpts["location"].(string)
	publisher, _ := cfg.ProviderOpts["publisher"].(string)

	// Expand env vars in project/location.
	var err error
	project, err = environment.Expand(ctx, project, env)
	if err != nil {
		return nil, fmt.Errorf("expanding project: %w", err)
	}
	location, err = environment.Expand(ctx, location, env)
	if err != nil {
		return nil, fmt.Errorf("expanding location: %w", err)
	}

	// Fall back to environment variables if not set in provider_opts.
	if project == "" {
		project, _ = env.Get(ctx, "GOOGLE_CLOUD_PROJECT")
	}
	if location == "" {
		location, _ = env.Get(ctx, "GOOGLE_CLOUD_LOCATION")
	}

	if project == "" {
		return nil, errors.New("vertex AI Model Garden requires a GCP project (set provider_opts.project or GOOGLE_CLOUD_PROJECT)")
	}
	if location == "" {
		return nil, errors.New("vertex AI Model Garden requires a GCP location (set provider_opts.location or GOOGLE_CLOUD_LOCATION)")
	}

	// Validate project and location to prevent URL path manipulation.
	if !validGCPIdentifier.MatchString(project) {
		return nil, fmt.Errorf("invalid GCP project ID: %q", project)
	}
	if !validGCPIdentifier.MatchString(location) {
		return nil, fmt.Errorf("invalid GCP location: %q", location)
	}

	// Build the base URL for the OpenAI-compatible endpoint.
	// https://cloud.google.com/vertex-ai/generative-ai/docs/partner-models/use-partner-models#openai_sdk
	baseURL := "https://" + location + "-aiplatform.googleapis.com/v1beta1/projects/" +
		url.PathEscape(project) + "/locations/" + url.PathEscape(location) + "/endpoints/openapi"

	slog.Debug("Creating Vertex AI Model Garden client",
		"publisher", publisher,
		"project", project,
		"location", location,
		"model", cfg.Model,
		"base_url", baseURL,
	)

	// Get a GCP access token using Application Default Credentials.
	tokenSource, err := google.DefaultTokenSource(ctx, cloudPlatformScope)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain GCP credentials for Vertex AI: %w (run 'gcloud auth application-default login')", err)
	}
	token, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get GCP access token: %w", err)
	}

	// Build a modified config that the OpenAI provider can use.
	// We override the base URL and set the token directly.
	oaiCfg := cfg.Clone()
	oaiCfg.BaseURL = baseURL
	// Use a synthetic token key env var — we'll set it in a wrapper env provider.
	const tokenEnvVar = "_VERTEX_AI_ACCESS_TOKEN"
	oaiCfg.TokenKey = tokenEnvVar

	// Remove provider_opts that are specific to Vertex AI / not relevant for OpenAI.
	delete(oaiCfg.ProviderOpts, "project")
	delete(oaiCfg.ProviderOpts, "location")
	delete(oaiCfg.ProviderOpts, "publisher")

	// Force chat completions API type (Vertex AI OpenAI endpoint uses this).
	if oaiCfg.ProviderOpts == nil {
		oaiCfg.ProviderOpts = map[string]any{}
	}
	oaiCfg.ProviderOpts["api_type"] = "openai_chatcompletions"

	// Wrap the environment provider to inject the GCP access token.
	wrappedEnv := &tokenEnv{
		Provider: env,
		key:      tokenEnvVar,
		tok:      token.AccessToken,
		ts:       tokenSource,
	}

	return openai.NewClient(ctx, oaiCfg, wrappedEnv, opts...)
}

// tokenEnv wraps an environment.Provider to inject a GCP access token.
// It refreshes the token on each Get call to handle token expiry.
type tokenEnv struct {
	environment.Provider

	key string
	mu  sync.Mutex
	tok string
	ts  oauth2.TokenSource
}

func (e *tokenEnv) Get(ctx context.Context, name string) (string, bool) {
	if name == e.key {
		e.mu.Lock()
		defer e.mu.Unlock()

		// Refresh token if needed — TokenSource handles caching.
		tok, err := e.ts.Token()
		if err != nil {
			slog.Warn("Failed to refresh GCP access token, using cached", "error", err)
			return e.tok, true
		}
		e.tok = tok.AccessToken
		return e.tok, true
	}
	return e.Provider.Get(ctx, name)
}

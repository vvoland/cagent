package config

import (
	"context"

	"github.com/docker/cagent/pkg/config/latest"
	"github.com/docker/cagent/pkg/environment"
)

var DefaultModels = map[string]string{
	"openai":    "gpt-5-mini",
	"anthropic": "claude-sonnet-4-0",
	"google":    "gemini-2.5-flash",
	"dmr":       "ai/qwen3:latest",
	"mistral":   "mistral-small-latest",
}

func AvailableProviders(ctx context.Context, modelsGateway string, env environment.Provider) []string {
	if modelsGateway != "" {
		// Default to anthropic when using a gateway
		return []string{"anthropic"}
	}

	var providers []string

	if key, _ := env.Get(ctx, "ANTHROPIC_API_KEY"); key != "" {
		providers = append(providers, "anthropic")
	}
	if key, _ := env.Get(ctx, "OPENAI_API_KEY"); key != "" {
		providers = append(providers, "openai")
	}
	if key, _ := env.Get(ctx, "GOOGLE_API_KEY"); key != "" {
		providers = append(providers, "google")
	}
	if key, _ := env.Get(ctx, "MISTRAL_API_KEY"); key != "" {
		providers = append(providers, "mistral")
	}

	providers = append(providers, "dmr")

	return providers
}

func AutoModelConfig(ctx context.Context, modelsGateway string, env environment.Provider) latest.ModelConfig {
	availableProviders := AvailableProviders(ctx, modelsGateway, env)
	firstAvailable := availableProviders[0]

	return latest.ModelConfig{
		Provider:  firstAvailable,
		Model:     DefaultModels[firstAvailable],
		MaxTokens: PreferredMaxTokens(firstAvailable),
	}
}

func PreferredMaxTokens(provider string) *int64 {
	var mt int64 = 32000
	if provider == "dmr" {
		mt = 16000
	}
	return &mt
}

// AutoEmbeddingModelConfigs returns the ordered list of embedding-capable models
// to try when a RAG strategy uses `model: auto` for embeddings.
//
// The priority is:
//  1. OpenAI -> text-embedding-3-small model
//  2. DMR -> Google's embeddinggemma model (via Docker Model Runner)
func AutoEmbeddingModelConfigs() []latest.ModelConfig {
	return []latest.ModelConfig{
		{
			Provider: "openai",
			Model:    "text-embedding-3-small",
		},
		{
			Provider: "dmr",
			Model:    "ai/embeddinggemma",
		},
	}
}

package dmr

import (
	"log/slog"
	"testing"

	latest "github.com/docker/cagent/pkg/config/v1"
)

func TestNewClientWithDefaultBaseURL(t *testing.T) {
	// Test case 1: No base_url provided, should use default
	cfg := &latest.ModelConfig{
		Provider: "dmr",
		Model:    "ai/qwen3",
		// BaseURL is empty, should use default
	}

	logger := slog.Default()
	client, err := NewClient(cfg, logger)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if client.baseURL != "http://localhost:12434/engines/llama.cpp/v1" {
		t.Errorf("Expected default baseURL to be 'http://localhost:12434/engines/llama.cpp/v1', got '%s'", client.baseURL)
	}
}

func TestNewClientWithExplicitBaseURL(t *testing.T) {
	// Test case 2: Explicit base_url provided, should use that
	customURL := "http://custom.example.com:8080/api/v1"
	cfg := &latest.ModelConfig{
		Provider: "dmr",
		Model:    "ai/qwen3",
		BaseURL:  customURL,
	}

	logger := slog.Default()
	client, err := NewClient(cfg, logger)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if client.baseURL != customURL {
		t.Errorf("Expected baseURL to be '%s', got '%s'", customURL, client.baseURL)
	}
}

func TestNewClientWithWrongType(t *testing.T) {
	// Test case 3: Wrong model type, should return error
	cfg := &latest.ModelConfig{
		Provider: "openai", // Wrong type
		Model:    "gpt-4",
	}

	logger := slog.Default()
	_, err := NewClient(cfg, logger)
	if err == nil {
		t.Fatal("Expected error for wrong model type, got nil")
	}
}

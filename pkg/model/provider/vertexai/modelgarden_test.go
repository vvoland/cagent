package vertexai

import (
	"testing"

	"github.com/docker/docker-agent/pkg/config/latest"
)

func TestIsModelGardenConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  *latest.ModelConfig
		want bool
	}{
		{
			name: "nil config",
			cfg:  nil,
			want: false,
		},
		{
			name: "no provider_opts",
			cfg:  &latest.ModelConfig{Provider: "google", Model: "gemini-2.5-flash"},
			want: false,
		},
		{
			name: "no publisher",
			cfg: &latest.ModelConfig{
				Provider:     "google",
				Model:        "gemini-2.5-flash",
				ProviderOpts: map[string]any{"project": "my-project", "location": "us-central1"},
			},
			want: false,
		},
		{
			name: "publisher=google",
			cfg: &latest.ModelConfig{
				Provider:     "google",
				Model:        "gemini-2.5-flash",
				ProviderOpts: map[string]any{"project": "my-project", "location": "us-central1", "publisher": "google"},
			},
			want: false,
		},
		{
			name: "publisher=anthropic",
			cfg: &latest.ModelConfig{
				Provider:     "google",
				Model:        "claude-sonnet-4-20250514",
				ProviderOpts: map[string]any{"project": "my-project", "location": "us-east5", "publisher": "anthropic"},
			},
			want: true,
		},
		{
			name: "publisher=meta",
			cfg: &latest.ModelConfig{
				Provider:     "google",
				Model:        "meta/llama-4-maverick-17b-128e-instruct-maas",
				ProviderOpts: map[string]any{"project": "my-project", "location": "us-central1", "publisher": "meta"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsModelGardenConfig(tt.cfg)
			if got != tt.want {
				t.Errorf("IsModelGardenConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidGCPIdentifier(t *testing.T) {
	valid := []string{"my-project", "us-central1", "project123", "ab"}
	for _, s := range valid {
		if !validGCPIdentifier.MatchString(s) {
			t.Errorf("expected %q to be valid", s)
		}
	}

	invalid := []string{"", "A", "../foo", "my project", "a", "123abc", "my_project/../../evil"}
	for _, s := range invalid {
		if validGCPIdentifier.MatchString(s) {
			t.Errorf("expected %q to be invalid", s)
		}
	}
}

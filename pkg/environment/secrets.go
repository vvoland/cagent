package environment

import (
	"context"
	"log/slog"
	"os"

	"github.com/docker/cagent/pkg/path"
)

type RunSecretsProvider struct {
	root string
}

func NewRunSecretsProvider() *RunSecretsProvider {
	return &RunSecretsProvider{
		root: "/run/secrets",
	}
}

func (p *RunSecretsProvider) Get(_ context.Context, name string) (string, bool) {
	// Validate the secret name to prevent path traversal
	validatedPath, err := path.ValidatePathInDirectory(name, p.root)
	if err != nil {
		slog.Debug("Invalid secret name", "name", name, "error", err)
		return "", false
	}

	buf, err := os.ReadFile(validatedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false
		}

		// Ignore error
		slog.Debug("Failed to find secret in /run/secrets", "error", err)
		return "", false
	}

	return string(buf), true
}

package environment

import (
	"context"
	"log/slog"
	"os"
	"strings"

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

func (p *RunSecretsProvider) Get(_ context.Context, name string) string {
	// Validate the secret name to prevent path traversal
	validatedPath, err := path.ValidatePathInDirectory(name, p.root)
	if err != nil {
		slog.Debug("Invalid secret name", "name", name, "error", err)
		return ""
	}

	buf, err := os.ReadFile(validatedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}

		// Ignore error
		slog.Debug("Failed to find secret in /run/secrets", "error", err)
		return ""
	}

	return strings.Split(string(buf), "\n")[0]
}

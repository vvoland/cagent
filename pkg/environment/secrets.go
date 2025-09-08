package environment

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
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
	path := filepath.Join(p.root, name)

	buf, err := os.ReadFile(path)
	if err != nil {
		// Ignore error
		slog.Debug("Failed to find secret in /run/secrets", "error", err)
		return ""
	}

	return strings.Split(string(buf), "\n")[0]
}

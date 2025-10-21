package mcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/goccy/go-yaml"

	"github.com/docker/cagent/pkg/environment"
	"github.com/docker/cagent/pkg/gateway"
	"github.com/docker/cagent/pkg/tools"
)

type GatewayToolset struct {
	cmdToolset *Toolset
	cleanUp    func() error
}

var _ tools.ToolSet = (*GatewayToolset)(nil)

func NewGatewayToolset(ctx context.Context, mcpServerName string, config any, envProvider environment.Provider) (*GatewayToolset, error) {
	slog.Debug("Creating MCP Gateway toolset", "name", mcpServerName)

	// Check which secrets (env vars) are required by the MCP server.
	secrets, err := gateway.RequiredEnvVars(ctx, mcpServerName)
	if err != nil {
		return nil, fmt.Errorf("reading which secrets the MCP server needs: %w", err)
	}

	// Make sure all the required secrets are available in the environment.
	// TODO(dga): Ideally, the MCP gateway would use the same provider that we have.
	fileSecrets, err := writeSecretsToFile(ctx, mcpServerName, secrets, envProvider)
	if err != nil {
		return nil, fmt.Errorf("writing secrets to file: %w", err)
	}

	fileConfig, err := writeConfigToFile(ctx, mcpServerName, config)
	if err != nil {
		os.Remove(fileSecrets)
		return nil, fmt.Errorf("writing config to file: %w", err)
	}

	// Isolate ourselves from the MCP Toolkit config by always using the Docker MCP catalog and custom config and secrets.
	// This improves shareability of agents.
	args := []string{
		"mcp", "gateway", "run",
		"--servers", mcpServerName,
		"--catalog", gateway.DockerCatalogURL,
		"--secrets", fileSecrets,
		"--config", fileConfig,
	}

	return &GatewayToolset{
		cmdToolset: NewToolsetCommand("docker", args, nil),
		cleanUp: func() error {
			return errors.Join(os.Remove(fileSecrets), os.Remove(fileConfig))
		},
	}, nil
}

func (t *GatewayToolset) Instructions() string {
	return t.cmdToolset.Instructions()
}

func (t *GatewayToolset) Tools(ctx context.Context) ([]tools.Tool, error) {
	return t.cmdToolset.Tools(ctx)
}

func (t *GatewayToolset) Start(ctx context.Context) error {
	return t.cmdToolset.Start(ctx)
}

func (t *GatewayToolset) SetElicitationHandler(handler tools.ElicitationHandler) {
	t.cmdToolset.SetElicitationHandler(handler)
}

func (t *GatewayToolset) SetOAuthSuccessHandler(handler func()) {
	t.cmdToolset.SetOAuthSuccessHandler(handler)
}

func (t *GatewayToolset) Stop(ctx context.Context) error {
	return errors.Join(t.cmdToolset.Stop(ctx), t.cleanUp())
}

func writeSecretsToFile(ctx context.Context, mcpServerName string, secrets []gateway.Secret, envProvider environment.Provider) (string, error) {
	var secretValues []string
	for _, secret := range secrets {
		v := envProvider.Get(ctx, secret.Env)
		if v == "" {
			return "", errors.New("missing environment variable " + secret.Env + " required by MCP server " + mcpServerName)
		}

		secretValues = append(secretValues, fmt.Sprintf("%s=%s", secret.Name, v))
	}

	// We have all the secrets, let's create a file with all of them for the MCP Gateway
	return writeTempFile("mcp-secrets-*", []byte(strings.Join(secretValues, "\n")))
}

func writeConfigToFile(_ context.Context, mcpServerName string, config any) (string, error) {
	buf, err := yaml.Marshal(map[string]any{
		mcpServerName: config,
	})
	if err != nil {
		return "", err
	}

	return writeTempFile("mcp-config-*", buf)
}

func writeTempFile(nameTemplate string, content []byte) (string, error) {
	f, err := os.CreateTemp("", nameTemplate)
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(content); err != nil {
		return "", err
	}

	return f.Name(), nil
}

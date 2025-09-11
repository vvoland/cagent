package oci

import (
	"bytes"
	"context"
	_ "embed"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/secrets"
)

//go:embed Dockerfile.template
var dockerfileTemplate string

func BuildDockerImage(ctx context.Context, agentFilePath, dockerImageName string, push bool) error {
	agentYaml, err := os.ReadFile(agentFilePath)
	if err != nil {
		return err
	}

	fileName := filepath.Base(agentFilePath)
	parentDir := filepath.Dir(agentFilePath)
	cfg, err := config.LoadConfigSecure(fileName, parentDir)
	if err != nil {
		return err
	}

	// Analyze the config to find which secrets are needed
	modelSecrets := secrets.GatherEnvVarsForModels(cfg)
	mcpServers := config.GatherMCPServerReferences(cfg)

	// Generate the Dockerfile
	var dockerfileBuf bytes.Buffer

	tpl := template.Must(template.New("Dockerfile").Parse(dockerfileTemplate))
	if err := tpl.Execute(&dockerfileBuf, map[string]any{
		"AgentConfig": string(agentYaml),
		"Description": cfg.Agents["root"].Description,
		"Licenses":    cfg.Metadata.License,
		"McpServers":  strings.Join(mcpServers, ","),
		"Secrets":     strings.Join(modelSecrets, ","),
	}); err != nil {
		return err
	}

	dockerfile := dockerfileBuf.String()
	slog.Debug("Generated Dockerfile", "dockerfile", dockerfile)

	// Run docker build
	buildArgs := []string{"build"}
	if dockerImageName != "" {
		buildArgs = append(buildArgs, "-t", dockerImageName)
		if push {
			buildArgs = append(buildArgs, "--push")
		}
	}
	buildArgs = append(buildArgs, "-")

	buildCmd := exec.CommandContext(ctx, "docker", buildArgs...)
	buildCmd.Stdin = strings.NewReader(dockerfile)
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	slog.Debug("running docker build", "args", buildArgs)

	return buildCmd.Run()
}

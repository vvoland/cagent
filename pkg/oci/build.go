package oci

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/secrets"
)

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
	secrets := secrets.GatherEnvVarsForModels(cfg)
	mcpServers := config.GatherMCPServerReferences(cfg)

	// TODO(dga): set the right entrypoint.
	dockerfile := fmt.Sprintf(`# syntax=docker/dockerfile:1
FROM alpine:3.22@sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1

RUN adduser -D cagent
ADD https://github.com/docker/cagent/releases/download/v1.0.9/cagent-linux-arm64 /cagent
RUN chmod +x /cagent
RUN cat <<EOF > /agent.yaml
%s
EOF
RUN chmod +r /agent.yaml
USER cagent
ENTRYPOINT ["/cagent", "run", "--debug", "--tui=false", "/agent.yaml", "get my username on github"]

LABEL com.docker.agent.packaging.version="v0.0.1"
LABEL com.docker.agent.runtime="cagent"
LABEL org.opencontainers.image.description="%s"
LABEL org.opencontainers.image.licenses="%s"
LABEL com.docker.agent.mcp-servers="%s"
LABEL com.docker.agent.secrets="%s"
`, string(agentYaml), cfg.Agents["root"].Description, cfg.Metadata.License, strings.Join(mcpServers, ","), strings.Join(secrets, ","))

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

	return buildCmd.Run()
}

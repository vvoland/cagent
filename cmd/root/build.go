package root

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/docker/cagent/internal/telemetry"
	"github.com/docker/cagent/pkg/config"
	latest "github.com/docker/cagent/pkg/config/v2"
	"github.com/docker/cagent/pkg/model/provider"
)

var push bool

func NewBuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "build <agent-file> <image-name>",
		Args:   cobra.ExactArgs(2),
		RunE:   runBuildCommand,
		Hidden: true,
	}

	cmd.PersistentFlags().BoolVar(&push, "push", false, "push the image")

	return cmd
}

func runBuildCommand(cmd *cobra.Command, args []string) error {
	telemetry.TrackCommand("build", args)

	fileName := filepath.Base(args[0])
	parentDir := filepath.Dir(args[0])

	cfg, err := config.LoadConfigSecure(fileName, parentDir)
	if err != nil {
		return err
	}

	secrets := gatherRequiredEnv(cfg)
	mcpServers := gatherMCPServers(cfg)

	tmp, err := os.MkdirTemp("", "build")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	// TODO(dga): set the right entrypoint.
	err = os.WriteFile(filepath.Join(tmp, "Dockerfile"), fmt.Appendf(nil, `# syntax=docker/dockerfile:1
FROM alpine:3.22@sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1

RUN adduser -D cagent
ADD https://github.com/docker/cagent/releases/download/v1.0.9/cagent-linux-arm64 /cagent
RUN chmod +x /cagent
COPY agent.yaml /
RUN chmod 666 /agent.yaml
USER cagent
ENTRYPOINT ["/cagent", "run", "--debug", "--tui=false", "/agent.yaml", "get my username on github"]

LABEL com.docker.agent.packaging.version="v0.0.1"
LABEL com.docker.agent.runtime="cagent"
LABEL org.opencontainers.image.description="%s"
LABEL org.opencontainers.image.licenses="%s"
LABEL com.docker.agent.mcp-servers="%s"
LABEL com.docker.agent.secrets="%s"
`, cfg.Agents["root"].Description, cfg.Metadata.License, strings.Join(mcpServers, ","), strings.Join(secrets, ",")), 0o700)
	if err != nil {
		return err
	}

	agentYaml, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(tmp, "agent.yaml"), agentYaml, 0o700)
	if err != nil {
		return err
	}

	buildArgs := []string{"build", "-t", args[1]}
	if push {
		buildArgs = append(buildArgs, "--push")
	}
	buildArgs = append(buildArgs, tmp)
	buildCmd := exec.CommandContext(cmd.Context(), "docker", buildArgs...)
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	return buildCmd.Run()
}

func gatherRequiredEnv(cfg *latest.Config) []string {
	requiredEnv := map[string]bool{}

	for name := range cfg.Models {
		model := cfg.Models[name]
		// Use the token environment variable from the alias if available
		if alias, exists := provider.ProviderAliases[model.Provider]; exists {
			if alias.TokenEnvVar != "" {
				requiredEnv[alias.TokenEnvVar] = true
			}
		} else {
			// Fallback to hardcoded mappings for unknown providers
			switch model.Provider {
			case "openai":
				requiredEnv["OPENAI_API_KEY"] = true
			case "anthropic":
				requiredEnv["ANTHROPIC_API_KEY"] = true
			case "google":
				requiredEnv["GOOGLE_API_KEY"] = true
			}
		}
	}

	for _, agent := range cfg.Agents {
		model := agent.Model
		switch {
		case strings.HasPrefix(model, "openai/"):
			requiredEnv["OPENAI_API_KEY"] = true
		case strings.HasPrefix(model, "anthropic/"):
			requiredEnv["ANTHROPIC_API_KEY"] = true
		case strings.HasPrefix(model, "google/"):
			requiredEnv["GOOGLE_API_KEY"] = true
		}
	}

	var requiredEnvList []string
	for e := range requiredEnv {
		requiredEnvList = append(requiredEnvList, e)
	}

	return requiredEnvList
}

func gatherMCPServers(cfg *latest.Config) []string {
	requiredServers := map[string]bool{}

	for _, agent := range cfg.Agents {
		for i := range agent.Toolsets {
			toolSet := agent.Toolsets[i]

			if toolSet.Type == "mcp" && toolSet.Ref != "" {
				requiredServers[toolSet.Ref] = true
			}
		}
	}

	var requiredServersList []string
	for e := range requiredServers {
		requiredServersList = append(requiredServersList, e)
	}

	return requiredServersList
}

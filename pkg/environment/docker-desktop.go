package environment

import (
	"context"

	"github.com/docker/cagent/pkg/desktop"
)

const (
	DockerDesktopEmail    = "DOCKER_EMAIL"
	DockerDesktopUsername = "DOCKER_USERNAME"
	DockerDesktopTokenEnv = "DOCKER_TOKEN"
)

type DockerDesktopProvider struct{}

func NewDockerDesktopProvider() *DockerDesktopProvider {
	return &DockerDesktopProvider{}
}

func (p *DockerDesktopProvider) Get(ctx context.Context, name string) string {
	switch name {
	case DockerDesktopEmail:
		return desktop.GetUserInfo(ctx).Email

	case DockerDesktopUsername:
		return desktop.GetUserInfo(ctx).Username

	case DockerDesktopTokenEnv:
		return desktop.GetToken(ctx)

	default:
		return ""
	}
}

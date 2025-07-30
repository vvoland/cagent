package auth

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/docker/cagent/pkg/desktop"
	"github.com/docker/cagent/pkg/environment"
)

func Token(ctx context.Context, env environment.Provider, baseURL, defaultEnvName string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	// Docker AI Gateway
	if strings.HasSuffix(u.Hostname(), ".dckr.io") {
		token := desktop.GetToken(ctx)
		if token == "" {
			return "", errors.New("sorry, you first need to sign in Docker Desktop to use the Docker AI Gateway")
		}
		return token, nil
	}

	// No Gateway
	token, err := env.Get(ctx, defaultEnvName)
	if err != nil || token == "" {
		return "", errors.New(defaultEnvName + " environment variable is required")
	}
	return token, nil
}

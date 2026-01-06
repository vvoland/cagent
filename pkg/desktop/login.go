package desktop

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type DockerHubInfo struct {
	Username string `json:"id"`
	Email    string `json:"email,omitempty"`
}

func GetToken(ctx context.Context) string {
	token := fetchToken(ctx)

	if token == "" || !isTokenExpired(token) {
		return token
	}

	if os.Getenv("EXPERIMENTAL_DOCKER_TOKEN_REFRESH") != "1" {
		return token
	}

	slog.Debug("Token expired, attempting docker login to refresh")
	if err := runDockerLogin(ctx); err != nil {
		slog.Debug("docker login failed", "error", err)
		return token
	}

	slog.Debug("docker login succeeded, fetching new token")
	return fetchToken(ctx)
}

func fetchToken(ctx context.Context) string {
	var token string
	_ = ClientBackend.Get(ctx, "/registry/token", &token)
	return token
}

func isTokenExpired(token string) bool {
	parsed, _, err := jwt.NewParser().ParseUnverified(token, jwt.MapClaims{})
	if err != nil {
		slog.Debug("Failed to parse JWT", "error", err)
		return false
	}

	exp, err := parsed.Claims.GetExpirationTime()
	if err != nil || exp == nil {
		slog.Debug("Failed to get expiration time from JWT", "error", err)
		return false
	}

	return exp.Before(time.Now())
}

func runDockerLogin(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "login")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func GetUserInfo(ctx context.Context) DockerHubInfo {
	var info DockerHubInfo
	_ = ClientBackend.Get(ctx, "/registry/username", &info)
	return info
}

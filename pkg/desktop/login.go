package desktop

import (
	"context"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type DockerHubInfo struct {
	Username string `json:"id"`
	Email    string `json:"email,omitempty"`
}

func GetToken(ctx context.Context) string {
	var token string
	_ = ClientBackend.Get(ctx, "/registry/token", &token)

	if token != "" {
		checkTokenExpiration(token)
	}

	return token
}

func checkTokenExpiration(token string) {
	// Parse the JWT without validation (we just need the claims)
	parsed, _, err := jwt.NewParser().ParseUnverified(token, jwt.MapClaims{})
	if err != nil {
		slog.Debug("Failed to parse JWT", "error", err)
		return
	}

	exp, err := parsed.Claims.GetExpirationTime()
	if err != nil {
		slog.Debug("Failed to get expiration time from JWT", "error", err)
		return
	}

	if exp == nil {
		slog.Debug("JWT has no expiration time")
		return
	}

	isExpired := exp.Before(time.Now())
	slog.Debug("JWT expiration check", "expiration", exp.Time, "expired", isExpired)
}

func GetUserInfo(ctx context.Context) DockerHubInfo {
	var info DockerHubInfo
	_ = ClientBackend.Get(ctx, "/registry/username", &info)
	return info
}

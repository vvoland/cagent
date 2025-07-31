package desktop

import (
	"context"
	"os"
	"sync"
)

type DockerHubInfo struct {
	Email         string   `json:"email,omitempty"`
	Organizations []string `json:"organizations,omitempty"`
	PlanName      string   `json:"planName"`
}

var (
	loginInfoCache     DockerHubInfo
	loginInfoCacheOnce sync.Once
)

func IsLoggedIn(ctx context.Context) bool {
	return GetToken(ctx) != ""
}

func GetToken(ctx context.Context) string {
	// Allow the user to override the token via an environment variable.
	// This is e.g useful when talking to a gateway on staging.
	manualToken := os.Getenv("DOCKER_TOKEN")
	if manualToken != "" {
		return manualToken
	}

	var token string
	_ = ClientBackend.Get(ctx, "/registry/token", &token)
	return token
}

func GetLoginInfo(ctx context.Context) DockerHubInfo {
	loginInfoCacheOnce.Do(func() {
		var info DockerHubInfo
		if err := ClientBackend.Get(ctx, "/registry/info", &info); err != nil {
			info = DockerHubInfo{}
		}

		loginInfoCache = info
	})
	return loginInfoCache
}

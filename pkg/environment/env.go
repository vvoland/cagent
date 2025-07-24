package environment

import (
	"context"
	"os"
	"strings"
)

type OsEnvProvider struct{}

func NewOsEnvProvider() *OsEnvProvider {
	return &OsEnvProvider{}
}

func (p *OsEnvProvider) Get(ctx context.Context, name string) (string, error) {
	return os.Getenv(name), nil
}

type KeyValueProvider struct {
	env map[string]string
}

func NewKeyValueProvider(env map[string]string) *KeyValueProvider {
	return &KeyValueProvider{
		env: env,
	}
}

func (p *KeyValueProvider) Get(ctx context.Context, name string) (string, error) {
	return expandEnv(p.env[name], os.Environ()), nil
}

func expandEnv(value string, env []string) string {
	return os.Expand(value, func(name string) string {
		for _, e := range env {
			if after, ok := strings.CutPrefix(e, name+"="); ok {
				return after
			}
		}
		return ""
	})
}

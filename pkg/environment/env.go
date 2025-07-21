package environment

import (
	"context"
	"os"
)

type EnvVariableProvider struct{}

func NewEnvVariableProvider() *EnvVariableProvider {
	return &EnvVariableProvider{}
}

func (p *EnvVariableProvider) Get(ctx context.Context, name string) (string, error) {
	return os.Getenv(name), nil
}

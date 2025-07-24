package environment

import (
	"context"
	"os"
)

type OsEnvProvider struct{}

func NewOsEnvProvider() *OsEnvProvider {
	return &OsEnvProvider{}
}

func (p *OsEnvProvider) Get(ctx context.Context, name string) (string, error) {
	return os.Getenv(name), nil
}

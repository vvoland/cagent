package environment

import (
	"context"
	"os"
)

// OsEnvProvider provides access to the operating system's environment variables.
type OsEnvProvider struct{}

func NewOsEnvProvider() *OsEnvProvider {
	return &OsEnvProvider{}
}

func (p *OsEnvProvider) Get(_ context.Context, name string) string {
	return os.Getenv(name)
}

type EnvFilesProviders struct {
	values []KeyValuePair
}

func NewEnvFilesProvider(absEnvFiles []string) (*EnvFilesProviders, error) {
	values, err := ReadEnvFiles(absEnvFiles)
	if err != nil {
		return nil, err
	}

	return &EnvFilesProviders{
		values: values,
	}, nil
}

func (p *EnvFilesProviders) Get(_ context.Context, name string) string {
	for _, kv := range p.values {
		if kv.Key == name {
			return kv.Value
		}
	}

	return ""
}

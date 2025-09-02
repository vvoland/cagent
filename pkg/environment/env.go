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

func (p *OsEnvProvider) Get(_ context.Context, name string) (string, error) {
	return os.Getenv(name), nil
}

// KeyValueProvider provides access to a map of key-value pairs as environment variables
// usually configured with `env:` section in a configuration file.
type KeyValueProvider struct {
	env map[string]string
}

func NewKeyValueProvider(env map[string]string) *KeyValueProvider {
	return &KeyValueProvider{
		env: env,
	}
}

func (p *KeyValueProvider) Get(_ context.Context, name string) (string, error) {
	return Expand(p.env[name], os.Environ()), nil
}

type EnvFilesProviders struct {
	absEnvFiles []string
}

func NewEnvFilesProvider(absEnvFiles []string) *EnvFilesProviders {
	return &EnvFilesProviders{
		absEnvFiles: absEnvFiles,
	}
}

func (p *EnvFilesProviders) Get(_ context.Context, name string) (string, error) {
	values, err := ReadEnvFiles(p.absEnvFiles)
	if err != nil {
		return "", err
	}

	for _, kv := range values {
		if kv.Key == name {
			return Expand(kv.Value, os.Environ()), nil
		}
	}

	return "", nil
}

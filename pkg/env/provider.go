package env

import "context"

type Provider interface {
	// GetEnv retrieves the value of an environment variable by name.
	GetEnv(ctx context.Context, name string) (string, error)
}

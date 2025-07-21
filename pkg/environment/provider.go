package environment

import "context"

type Provider interface {
	// Get retrieves the value of an environment variable by name.
	Get(ctx context.Context, name string) (string, error)
}

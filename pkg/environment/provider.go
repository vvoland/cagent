package environment

import "context"

type Provider interface {
	// Get retrieves the value of an environment variable by name.
	// Returns (value, true) if found (value may be empty).
	// Returns ("", false) if not found.
	Get(ctx context.Context, name string) (string, bool)
}

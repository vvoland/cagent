package env

import "context"

type NoFailProvider struct {
	provider Provider
}

func NewNoFailProvider(provider Provider) *NoFailProvider {
	return &NoFailProvider{
		provider: provider,
	}
}

func (p *NoFailProvider) GetEnv(ctx context.Context, name string) (string, error) {
	value, err := p.provider.GetEnv(ctx, name)
	if err != nil {
		// Ignore the error and return an empty string
		return "", nil
	}

	return value, nil
}

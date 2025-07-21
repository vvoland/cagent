package env

import "context"

type MultiProvider struct {
	providers []Provider
}

func NewMultiProvider(providers ...Provider) *MultiProvider {
	return &MultiProvider{
		providers: providers,
	}
}

func (p *MultiProvider) GetEnv(ctx context.Context, name string) (string, error) {
	for _, provider := range p.providers {
		value, err := provider.GetEnv(ctx, name)
		if err != nil {
			return "", err
		}

		if value != "" {
			return value, nil
		}
	}

	return "", nil
}

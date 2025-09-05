package environment

import "context"

type MultiProvider struct {
	providers []Provider
}

func NewMultiProvider(providers ...Provider) *MultiProvider {
	return &MultiProvider{
		providers: providers,
	}
}

func (p *MultiProvider) Get(ctx context.Context, name string) string {
	for _, provider := range p.providers {
		value := provider.Get(ctx, name)
		if value != "" {
			return value
		}
	}

	return ""
}

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

func (p *MultiProvider) Get(ctx context.Context, name string) (string, bool) {
	for _, provider := range p.providers {
		value, found := provider.Get(ctx, name)
		if found {
			return value, true
		}
	}

	return "", false
}

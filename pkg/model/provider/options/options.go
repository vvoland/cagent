package options

type ModelOptions struct {
	gateway string
}

func (c *ModelOptions) Gateway() string {
	return c.gateway
}

type Opt func(*ModelOptions)

func WithGateway(gateway string) Opt {
	return func(cfg *ModelOptions) {
		cfg.gateway = gateway
	}
}

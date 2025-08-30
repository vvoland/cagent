package environment

func NewDefaultProvider() Provider {
	return NewMultiProvider(
		NewOsEnvProvider(),
		NewNoFailProvider(
			NewOnePasswordProvider(),
		),
	)
}

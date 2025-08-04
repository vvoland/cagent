package environment

func NewDefaultProvider() Provider {
	p := []Provider{
		NewOsEnvProvider(),
		NewNoFailProvider(
			NewOnePasswordProvider(),
		),
	}

	passProvider, err := NewPassProvider()
	if err == nil {
		p = append(p, passProvider)
	}

	keychainProvider, err := NewKeychainProvider()
	if err == nil {
		p = append(p, keychainProvider)
	}

	return NewMultiProvider(p...)
}

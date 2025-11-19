package environment

func NewDefaultProvider() Provider {
	providers := []Provider{
		NewOsEnvProvider(),
		NewRunSecretsProvider(),
		NewDockerDesktopProvider(),
	}

	// Append pass provider at the end if available
	if passProvider, err := NewPassProvider(); err == nil {
		providers = append(providers, passProvider)
	}

	// Append keychain provider if available
	if keychainProvider, err := NewKeychainProvider(); err == nil {
		providers = append(providers, keychainProvider)
	}

	return NewMultiProvider(providers...)
}

package environment

import "log/slog"

func NewDefaultProvider(logger *slog.Logger) Provider {
	return NewMultiProvider(
		NewEnvVariableProvider(),
		NewNoFailProvider(
			NewOnePasswordProvider(logger),
		),
	)
}

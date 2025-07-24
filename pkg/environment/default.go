package environment

import "log/slog"

func NewDefaultProvider(logger *slog.Logger) Provider {
	return NewMultiProvider(
		NewOsEnvProvider(),
		NewNoFailProvider(
			NewOnePasswordProvider(logger),
		),
	)
}

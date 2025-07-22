package environment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/1password/onepassword-sdk-go"
)

type OnePasswordProvider struct {
	onceSecrets    sync.Once
	secrets        onepassword.SecretsAPI
	onceSecretsErr error
	logger         *slog.Logger
}

func NewOnePasswordProvider(logger *slog.Logger) *OnePasswordProvider {
	return &OnePasswordProvider{
		logger: logger,
	}
}

func (p *OnePasswordProvider) Get(ctx context.Context, name string) (string, error) {
	p.onceSecrets.Do(func() {
		opToken := os.Getenv("OP_SERVICE_ACCOUNT_TOKEN")
		if opToken == "" {
			p.onceSecretsErr = errors.New("OP_SERVICE_ACCOUNT_TOKEN environment variable is required for 1Password integration")
			return
		}

		path := "op://cagent/" + name + "/password"
		p.logger.Debug("Looking for environment variable in 1Password", "path", path)

		client, err := onepassword.NewClient(ctx,
			onepassword.WithServiceAccountToken(opToken),
			onepassword.WithIntegrationInfo("cagent 1Password Integration", "v1.0.0"),
		)
		if err != nil {
			p.onceSecretsErr = fmt.Errorf("failed to connect to 1Password: %w", err)
			return
		}

		p.secrets = client.Secrets()
	})

	if p.onceSecretsErr != nil {
		return "", p.onceSecretsErr
	}

	secret, err := p.secrets.Resolve(ctx, "op://cagent/"+name+"/password")
	if err != nil {
		return "", fmt.Errorf("failed to find environment variable (%s) in 1Password: %w", name, err)
	}

	return secret, nil
}

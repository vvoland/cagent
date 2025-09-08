package environment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/1password/onepassword-sdk-go"
)

type OnePasswordProvider struct {
	opToken string
}

func NewOnePasswordProvider(ctx context.Context) (*OnePasswordProvider, error) {
	opToken := os.Getenv("OP_SERVICE_ACCOUNT_TOKEN")
	if opToken == "" {
		return nil, errors.New("OP_SERVICE_ACCOUNT_TOKEN environment variable is required for 1Password integration")
	}

	_, err := onepassword.NewClient(ctx,
		onepassword.WithServiceAccountToken(opToken),
		onepassword.WithIntegrationInfo("cagent 1Password Integration", "v1.0.0"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to 1Password: %w", err)
	}

	return &OnePasswordProvider{
		opToken: opToken,
	}, nil
}

func (p *OnePasswordProvider) Get(ctx context.Context, name string) string {
	// This thing is not thread-safe, so we create a new client each time (for now)
	// even though it's probably too slow.
	client, _ := onepassword.NewClient(ctx,
		onepassword.WithServiceAccountToken(p.opToken),
		onepassword.WithIntegrationInfo("cagent 1Password Integration", "v1.0.0"),
	)

	secret, err := client.Secrets().Resolve(ctx, "op://cagent/"+name+"/password")
	if err != nil {
		// Ignore error
		slog.Debug("Failed to find secret in 1Password", "error", err)
		return ""
	}

	return secret
}

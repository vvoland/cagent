package environment

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
)

// KeychainProvider is a provider that retrieves secrets using the macOS keychain
// via the `security` command-line tool.
type KeychainProvider struct{}

type ErrKeychainNotAvailable struct{}

func (ErrKeychainNotAvailable) Error() string {
	return "security command is not available (macOS keychain access)"
}

// NewKeychainProvider creates a new KeychainProvider instance.
// It verifies that the `security` command is available on the system.
func NewKeychainProvider() (*KeychainProvider, error) {
	path, err := exec.LookPath("security")
	if err != nil && !errors.Is(err, exec.ErrNotFound) {
		slog.Warn("failed to lookup `security` binary", "error", err)
	}
	if path == "" {
		return nil, ErrKeychainNotAvailable{}
	}
	return &KeychainProvider{}, nil
}

// Get retrieves the value of a secret by its service name from the macOS keychain.
// It uses the `security find-generic-password -w -s <name>` command to fetch the password.
func (p *KeychainProvider) Get(ctx context.Context, name string) (string, error) {
	cmd := exec.CommandContext(ctx, "security", "find-generic-password", "-w", "-s", name)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to retrieve secret from keychain: %w, stderr: %v", err, stderr.String())
	}

	return strings.TrimSpace(out.String()), nil
}

package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/99designs/keyring"
)

const keyringServiceName = "docker-agent-oauth"

// KeyringTokenStore implements OAuthTokenStore using the OS-native credential store
// (macOS Keychain, Windows Credential Manager, Linux Secret Service).
type KeyringTokenStore struct {
	ring keyring.Keyring
}

// NewKeyringTokenStore creates a token store backed by the OS keychain.
// Falls back to InMemoryTokenStore if no keyring backend is available.
func NewKeyringTokenStore() OAuthTokenStore {
	ring, err := keyring.Open(keyring.Config{
		ServiceName:                    keyringServiceName,
		KeychainTrustApplication:       true,
		KeychainSynchronizable:         false,
		KeychainAccessibleWhenUnlocked: true,
	})
	if err != nil {
		slog.Warn("OS keyring not available, falling back to in-memory token store", "error", err)
		return NewInMemoryTokenStore()
	}

	// Validate the keyring is actually usable by attempting a get.
	// Some backends (e.g. file) open successfully but fail on operations.
	_, err = ring.Get("docker-agent-probe")
	if err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
		slog.Warn("OS keyring not usable, falling back to in-memory token store", "error", err)
		return NewInMemoryTokenStore()
	}

	return &KeyringTokenStore{ring: ring}
}

// keyringKey returns a stable key for a given resource URL.
func keyringKey(resourceURL string) string {
	return "oauth:" + resourceURL
}

func (s *KeyringTokenStore) GetToken(resourceURL string) (*OAuthToken, error) {
	item, err := s.ring.Get(keyringKey(resourceURL))
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return nil, fmt.Errorf("no token found for resource: %s", resourceURL)
		}
		return nil, fmt.Errorf("keyring get failed: %w", err)
	}

	var token OAuthToken
	if err := json.Unmarshal(item.Data, &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &token, nil
}

func (s *KeyringTokenStore) StoreToken(resourceURL string, token *OAuthToken) error {
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	return s.ring.Set(keyring.Item{
		Key:         keyringKey(resourceURL),
		Data:        data,
		Label:       "Docker Agent OAuth Token",
		Description: "OAuth token for " + resourceURL,
	})
}

func (s *KeyringTokenStore) RemoveToken(resourceURL string) error {
	err := s.ring.Remove(keyringKey(resourceURL))
	if err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
		return fmt.Errorf("keyring remove failed: %w", err)
	}
	return nil
}

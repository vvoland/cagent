package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/99designs/keyring"
)

const keyringServiceName = "docker-agent-oauth"

const indexKey = "oauth:_index"

// KeyringTokenStore implements OAuthTokenStore using the OS-native credential store
// (macOS Keychain, Windows Credential Manager, Linux Secret Service).
type KeyringTokenStore struct {
	ring keyring.Keyring
}

func openKeyring() (keyring.Keyring, error) {
	return keyring.Open(keyring.Config{
		ServiceName:                    keyringServiceName,
		KeychainTrustApplication:       true,
		KeychainSynchronizable:         false,
		KeychainAccessibleWhenUnlocked: true,
	})
}

// NewKeyringTokenStore creates a token store backed by the OS keychain.
// Falls back to InMemoryTokenStore if no keyring backend is available.
func NewKeyringTokenStore() OAuthTokenStore {
	ring, err := openKeyring()
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

	if err := s.ring.Set(keyring.Item{
		Key:         keyringKey(resourceURL),
		Data:        data,
		Label:       "Docker Agent OAuth Token",
		Description: "OAuth token for " + resourceURL,
	}); err != nil {
		return err
	}

	// Update the index
	return s.addToIndex(resourceURL)
}

func (s *KeyringTokenStore) RemoveToken(resourceURL string) error {
	err := s.ring.Remove(keyringKey(resourceURL))
	if err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
		return fmt.Errorf("keyring remove failed: %w", err)
	}

	// Update the index
	return s.removeFromIndex(resourceURL)
}

// loadIndex reads the resource URL index from the keyring.
func (s *KeyringTokenStore) loadIndex() ([]string, error) {
	return loadIndex(s.ring)
}

func loadIndex(ring keyring.Keyring) ([]string, error) {
	item, err := ring.Get(indexKey)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read index: %w", err)
	}

	var urls []string
	if err := json.Unmarshal(item.Data, &urls); err != nil {
		return nil, fmt.Errorf("failed to unmarshal index: %w", err)
	}
	return urls, nil
}

// saveIndex writes the resource URL index to the keyring.
func (s *KeyringTokenStore) saveIndex(urls []string) error {
	data, err := json.Marshal(urls)
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	return s.ring.Set(keyring.Item{
		Key:         indexKey,
		Data:        data,
		Label:       "Docker Agent OAuth Index",
		Description: "Index of OAuth resource URLs",
	})
}

func (s *KeyringTokenStore) addToIndex(resourceURL string) error {
	urls, err := s.loadIndex()
	if err != nil {
		return err
	}

	if slices.Contains(urls, resourceURL) {
		return nil // already present
	}

	return s.saveIndex(append(urls, resourceURL))
}

func (s *KeyringTokenStore) removeFromIndex(resourceURL string) error {
	urls, err := s.loadIndex()
	if err != nil {
		return err
	}

	filtered := urls[:0]
	for _, u := range urls {
		if u != resourceURL {
			filtered = append(filtered, u)
		}
	}

	if len(filtered) == 0 {
		// Remove the index key entirely if empty
		err := s.ring.Remove(indexKey)
		if err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
			return err
		}
		return nil
	}

	return s.saveIndex(filtered)
}

// OAuthTokenEntry represents a stored OAuth token along with its resource URL.
type OAuthTokenEntry struct {
	ResourceURL string
	Token       *OAuthToken
}

// ListOAuthTokens opens the OS keyring and returns all stored OAuth tokens.
// It reads a stored index to discover resource URLs, then fetches each token.
// This results in only Get() calls (no Keys()), so the user is prompted for
// keychain access at most once.
func ListOAuthTokens() ([]OAuthTokenEntry, error) {
	ring, err := openKeyring()
	if err != nil {
		return nil, fmt.Errorf("failed to open keyring: %w", err)
	}

	urls, err := loadIndex(ring)
	if err != nil {
		return nil, err
	}

	// Fall back to Keys() for tokens stored before the index existed.
	if len(urls) == 0 {
		urls, err = discoverResourceURLs(ring)
		if err != nil {
			return nil, err
		}
	}

	var entries []OAuthTokenEntry
	for _, resourceURL := range urls {
		item, err := ring.Get(keyringKey(resourceURL))
		if err != nil {
			if errors.Is(err, keyring.ErrKeyNotFound) {
				continue // stale index entry
			}
			slog.Warn("Failed to read keyring item", "resource", resourceURL, "error", err)
			continue
		}

		var token OAuthToken
		if err := json.Unmarshal(item.Data, &token); err != nil {
			slog.Warn("Failed to unmarshal token", "resource", resourceURL, "error", err)
			continue
		}

		entries = append(entries, OAuthTokenEntry{
			ResourceURL: resourceURL,
			Token:       &token,
		})
	}

	return entries, nil
}

// discoverResourceURLs uses Keys() to find oauth resource URLs.
// This is used as a fallback for tokens stored before the index was introduced.
func discoverResourceURLs(ring keyring.Keyring) ([]string, error) {
	keys, err := ring.Keys()
	if err != nil {
		return nil, fmt.Errorf("failed to list keyring keys: %w", err)
	}

	const prefix = "oauth:"
	var urls []string
	for _, key := range keys {
		if strings.HasPrefix(key, prefix) && key != indexKey {
			urls = append(urls, strings.TrimPrefix(key, prefix))
		}
	}
	return urls, nil
}

// RemoveOAuthToken opens the OS keyring and removes the token for the given resource URL.
func RemoveOAuthToken(resourceURL string) error {
	store := NewKeyringTokenStore()
	krs, ok := store.(*KeyringTokenStore)
	if !ok {
		return errors.New("OS keyring not available")
	}
	return krs.RemoveToken(resourceURL)
}

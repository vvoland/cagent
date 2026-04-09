package mcp

import (
	"encoding/json"
	"testing"
	"time"
)

func TestKeyringTokenStore_RoundTrip(t *testing.T) {
	// Use in-memory fallback for CI environments without a keyring.
	// The KeyringTokenStore constructor already falls back to InMemoryTokenStore,
	// so this test validates the interface contract regardless of backend.
	store := NewKeyringTokenStore()

	resourceURL := "https://example.com/mcp"

	// Initially no token
	_, err := store.GetToken(resourceURL)
	if err == nil {
		t.Fatal("expected error for missing token")
	}

	// Store a token
	token := &OAuthToken{
		AccessToken:  "access-123",
		TokenType:    "Bearer",
		RefreshToken: "refresh-456",
		ExpiresIn:    3600,
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}
	if err := store.StoreToken(resourceURL, token); err != nil {
		t.Fatalf("StoreToken failed: %v", err)
	}

	// Retrieve it
	got, err := store.GetToken(resourceURL)
	if err != nil {
		t.Fatalf("GetToken failed: %v", err)
	}
	if got.AccessToken != "access-123" {
		t.Errorf("AccessToken = %q, want %q", got.AccessToken, "access-123")
	}
	if got.RefreshToken != "refresh-456" {
		t.Errorf("RefreshToken = %q, want %q", got.RefreshToken, "refresh-456")
	}

	// Remove it
	if err := store.RemoveToken(resourceURL); err != nil {
		t.Fatalf("RemoveToken failed: %v", err)
	}

	_, err = store.GetToken(resourceURL)
	if err == nil {
		t.Fatal("expected error after RemoveToken")
	}
}

func TestKeyringTokenStore_JSONRoundTrip(t *testing.T) {
	// Verify that OAuthToken serializes correctly (important for keyring storage)
	token := &OAuthToken{
		AccessToken:  "at",
		TokenType:    "Bearer",
		RefreshToken: "rt",
		ExpiresIn:    7200,
		ExpiresAt:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Scope:        "read write",
	}

	data, err := json.Marshal(token)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var got OAuthToken
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if got.AccessToken != token.AccessToken || got.RefreshToken != token.RefreshToken || got.Scope != token.Scope {
		t.Errorf("JSON round-trip mismatch: got %+v, want %+v", got, token)
	}
}

func TestKeyringTokenStore_RemoveNonExistent(t *testing.T) {
	store := NewKeyringTokenStore()
	// Should not error when removing a non-existent token
	if err := store.RemoveToken("https://nonexistent.example.com"); err != nil {
		t.Fatalf("RemoveToken for non-existent key should not error: %v", err)
	}
}

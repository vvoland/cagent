package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/client/transport"
)

// AuthServerMetadata represents the OAuth 2.0 Authorization Server Metadata
// This is a copy from the mcp-go library to avoid importing internal types
type AuthServerMetadata struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	RegistrationEndpoint              string   `json:"registration_endpoint,omitempty"`
	JwksURI                           string   `json:"jwks_uri,omitempty"`
	ScopesSupported                   []string `json:"scopes_supported,omitempty"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported,omitempty"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported,omitempty"`
}

// OAuthProtectedResource represents the response from /.well-known/oauth-protected-resource
type OAuthProtectedResource struct {
	AuthorizationServers []string `json:"authorization_servers"`
	Resource             string   `json:"resource"`
	ResourceName         string   `json:"resource_name,omitempty"`
}

// GetAuthorizationURL gets the OAuth authorization URL using the correct discovery logic
func GetAuthorizationURL(ctx context.Context, oauthHandler *transport.OAuthHandler, state, codeChallenge string) (string, error) {
	// Get server metadata using our corrected discovery logic
	metadata, err := getCorrectServerMetadata(ctx, oauthHandler)
	if err != nil {
		return "", fmt.Errorf("failed to get server metadata: %w", err)
	}

	// Store the state for later validation
	oauthHandler.SetExpectedState(state)

	// Use the handler's existing logic to build the URL parameters correctly
	// but replace the base URL with our corrected authorization endpoint
	originalURL, err := oauthHandler.GetAuthorizationURL(ctx, state, codeChallenge)
	if err != nil {
		return "", fmt.Errorf("failed to get authorization URL from handler: %w", err)
	}

	// Parse the original URL to extract parameters
	parsedURL, err := url.Parse(originalURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse authorization URL: %w", err)
	}

	// Replace the base URL with our corrected authorization endpoint
	return metadata.AuthorizationEndpoint + "?" + parsedURL.RawQuery, nil
}

// getCorrectServerMetadata implements the correct OAuth discovery sequence:
// 1. Try /.well-known/oauth-protected-resource (RFC 9728)
// 2. If that fails, try /.well-known/oauth-authorization-server (RFC 8414)
// 3. If both fail, use default endpoints
func getCorrectServerMetadata(ctx context.Context, oauthHandler *transport.OAuthHandler) (*AuthServerMetadata, error) {
	// Extract base URL from the OAuth handler
	// We'll use the existing metadata call just to get the base URL, then do our own discovery
	baseURL, err := extractBaseURLFromHandler(ctx, oauthHandler)
	if err != nil {
		return nil, fmt.Errorf("failed to extract base URL: %w", err)
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}

	// Step 1: Try OAuth Protected Resource discovery (RFC 9728)
	protectedResourceURL := baseURL + "/.well-known/oauth-protected-resource"
	metadata, err := tryProtectedResourceDiscovery(ctx, httpClient, protectedResourceURL)
	if err == nil {
		return metadata, nil
	}

	// Step 2: Try OAuth Authorization Server Metadata discovery (RFC 8414)
	authServerURL := baseURL + "/.well-known/oauth-authorization-server"
	metadata, err = tryDirectMetadataDiscovery(ctx, httpClient, authServerURL)
	if err == nil {
		return metadata, nil
	}

	// Step 3: Try OpenID Connect discovery as additional fallback
	openidURL := baseURL + "/.well-known/openid-configuration"
	metadata, err = tryDirectMetadataDiscovery(ctx, httpClient, openidURL)
	if err == nil {
		return metadata, nil
	}

	// Step 4: Use default endpoints as final fallback
	return getDefaultEndpoints(baseURL), nil
}

// extractBaseURLFromHandler extracts the base URL from the OAuth handler
func extractBaseURLFromHandler(ctx context.Context, oauthHandler *transport.OAuthHandler) (string, error) {
	// We need to get the base URL somehow. The cleanest way is to call the existing
	// metadata method and extract the base URL from the result, even though the
	// discovery logic in that method is flawed.
	existingMetadata, err := oauthHandler.GetServerMetadata(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get base URL from handler: %w", err)
	}

	// Extract base URL from any endpoint (authorization endpoint is always available)
	parsedURL, err := url.Parse(existingMetadata.AuthorizationEndpoint)
	if err != nil {
		return "", fmt.Errorf("failed to parse authorization endpoint: %w", err)
	}

	// Build base URL without path
	return fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host), nil
}

// tryProtectedResourceDiscovery tries OAuth Protected Resource discovery (RFC 9728)
func tryProtectedResourceDiscovery(ctx context.Context, httpClient *http.Client, protectedResourceURL string) (*AuthServerMetadata, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, protectedResourceURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create protected resource request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("MCP-Protocol-Version", "2025-03-26")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send protected resource request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("protected resource discovery failed with status %d", resp.StatusCode)
	}

	var protectedResource OAuthProtectedResource
	if err := json.NewDecoder(resp.Body).Decode(&protectedResource); err != nil {
		return nil, fmt.Errorf("failed to decode protected resource response: %w", err)
	}

	// If no authorization servers are specified, this discovery method failed
	if len(protectedResource.AuthorizationServers) == 0 {
		return nil, fmt.Errorf("no authorization servers found in protected resource")
	}

	// Use the first authorization server and try to get its metadata
	authServerURL := protectedResource.AuthorizationServers[0]

	// Try OpenID Connect discovery first
	metadata, err := tryDirectMetadataDiscovery(ctx, httpClient, authServerURL+"/.well-known/openid-configuration")
	if err == nil {
		return metadata, nil
	}

	// If OpenID Connect discovery fails, try OAuth Authorization Server Metadata
	return tryDirectMetadataDiscovery(ctx, httpClient, authServerURL+"/.well-known/oauth-authorization-server")
}

// tryDirectMetadataDiscovery tries to fetch metadata directly from a discovery URL
func tryDirectMetadataDiscovery(ctx context.Context, httpClient *http.Client, metadataURL string) (*AuthServerMetadata, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metadataURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create metadata request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("MCP-Protocol-Version", "2025-03-26")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send metadata request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata discovery failed with status %d", resp.StatusCode)
	}

	var metadata AuthServerMetadata
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, fmt.Errorf("failed to decode metadata response: %w", err)
	}

	return &metadata, nil
}

// getDefaultEndpoints returns default OAuth endpoints based on the base URL
func getDefaultEndpoints(baseURL string) *AuthServerMetadata {
	// Parse the base URL to extract the authority
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		// If parsing fails, just use the baseURL as-is
		return &AuthServerMetadata{
			Issuer:                baseURL,
			AuthorizationEndpoint: baseURL + "/authorize",
			TokenEndpoint:         baseURL + "/token",
			RegistrationEndpoint:  baseURL + "/register",
		}
	}

	// Discard any path component to get the authorization base URL
	parsedURL.Path = ""
	authBaseURL := parsedURL.String()

	// Remove trailing slash if present
	authBaseURL = strings.TrimSuffix(authBaseURL, "/")

	return &AuthServerMetadata{
		Issuer:                authBaseURL,
		AuthorizationEndpoint: authBaseURL + "/authorize",
		TokenEndpoint:         authBaseURL + "/token",
		RegistrationEndpoint:  authBaseURL + "/register",
	}
}

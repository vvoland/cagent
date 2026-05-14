// Package chatgpt implements OAuth authentication for ChatGPT Plus/Pro subscriptions.
// It uses the OAuth2 PKCE flow against auth.openai.com to obtain access tokens
// that can be exchanged for an OpenAI API key.
package chatgpt

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/docker/docker-agent/pkg/browser"
)

const (
	authorizationEndpoint = "https://auth.openai.com/oauth/authorize"

	clientID = "app_EMoamEEZ73f0CkXaXp7hrann"

	defaultScopes = "openid profile email offline_access"

	defaultPort = 1455
)

// tokenEndpointURL is the OAuth token endpoint. It is a variable so tests can override it.
var tokenEndpointURL = "https://auth.openai.com/oauth/token"

// Token represents the persisted authentication state from the ChatGPT OAuth flow.
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	IDToken      string    `json:"id_token,omitempty"`
	AccountID    string    `json:"account_id,omitempty"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// IsExpired returns true if the token will expire within 60 seconds.
func (t *Token) IsExpired() bool {
	if t.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().Add(60 * time.Second).After(t.ExpiresAt)
}

// Login performs the OAuth PKCE flow to authenticate with ChatGPT.
// It opens the user's browser to the OpenAI login page, starts a local
// callback server, and exchanges the authorization code for tokens.
func Login(ctx context.Context) (*Token, error) {
	verifier := oauth2.GenerateVerifier()
	challenge := oauth2.S256ChallengeFromVerifier(verifier)

	state, err := generateState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	var lc net.ListenConfig
	listener, err := lc.Listen(ctx, "tcp", fmt.Sprintf("127.0.0.1:%d", defaultPort))
	if err != nil {
		listener, err = lc.Listen(ctx, "tcp", "127.0.0.1:0")
		if err != nil {
			return nil, fmt.Errorf("failed to start callback server: %w", err)
		}
	}
	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://localhost:%d/auth/callback", port)

	authURL := buildAuthURL(redirectURI, state, challenge)

	slog.Debug("Starting ChatGPT OAuth login", "redirect_uri", redirectURI)

	type callbackResult struct {
		code string
		err  error
	}
	resultCh := make(chan callbackResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		if r.URL.Query().Get("state") != state {
			resultCh <- callbackResult{err: errors.New("state mismatch")}
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}

		if errParam := r.URL.Query().Get("error"); errParam != "" {
			desc := r.URL.Query().Get("error_description")
			resultCh <- callbackResult{err: fmt.Errorf("OAuth error: %s: %s", errParam, desc)}
			fmt.Fprintf(w, "<html><body><h1>Authentication failed</h1><p>%s</p><p>You can close this window.</p></body></html>", html.EscapeString(desc))
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			resultCh <- callbackResult{err: errors.New("no authorization code received")}
			http.Error(w, "No code received", http.StatusBadRequest)
			return
		}

		resultCh <- callbackResult{code: code}
		fmt.Fprint(w, "<html><body><h1>Authentication successful!</h1><p>You can close this window and return to the terminal.</p></body></html>")
	})

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			slog.Error("Callback server error", "error", err)
		}
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if err := browser.Open(ctx, authURL); err != nil {
		return nil, fmt.Errorf("failed to open browser (visit this URL manually):\n%s\n\nerror: %w", authURL, err)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-resultCh:
		if result.err != nil {
			return nil, fmt.Errorf("authentication failed: %w", result.err)
		}

		tokens, err := exchangeCode(ctx, result.code, verifier, redirectURI)
		if err != nil {
			return nil, err
		}

		return tokens, nil
	}
}

// RefreshAccessToken refreshes an expired access token using the refresh token.
func RefreshAccessToken(ctx context.Context, refreshToken string) (*Token, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", clientID)
	data.Set("refresh_token", refreshToken)

	var refreshResp struct {
		IDToken      string `json:"id_token"`
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := postForm(ctx, tokenEndpointURL, data, &refreshResp); err != nil {
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}

	newRefreshToken := refreshResp.RefreshToken
	if newRefreshToken == "" {
		newRefreshToken = refreshToken
	}

	if refreshResp.IDToken == "" {
		return nil, errors.New("refresh response did not include an id_token")
	}

	accountID := extractAccountID(refreshResp.IDToken, refreshResp.AccessToken)

	expiry := 1 * time.Hour
	if refreshResp.ExpiresIn > 0 {
		expiry = time.Duration(refreshResp.ExpiresIn) * time.Second
	}

	slog.Debug("ChatGPT token refreshed successfully")
	return &Token{
		AccessToken:  refreshResp.AccessToken,
		RefreshToken: newRefreshToken,
		IDToken:      refreshResp.IDToken,
		AccountID:    accountID,
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(expiry),
	}, nil
}

func buildAuthURL(redirectURI, state, codeChallenge string) string {
	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", clientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("scope", defaultScopes)
	params.Set("code_challenge", codeChallenge)
	params.Set("code_challenge_method", "S256")
	params.Set("state", state)
	return authorizationEndpoint + "?" + params.Encode()
}

func exchangeCode(ctx context.Context, code, verifier, redirectURI string) (*Token, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("client_id", clientID)
	data.Set("code_verifier", verifier)

	var tokenResp struct {
		IDToken      string `json:"id_token"`
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := postForm(ctx, tokenEndpointURL, data, &tokenResp); err != nil {
		return nil, fmt.Errorf("code exchange failed: %w", err)
	}

	accountID := extractAccountID(tokenResp.IDToken, tokenResp.AccessToken)

	expiry := 1 * time.Hour
	if tokenResp.ExpiresIn > 0 {
		expiry = time.Duration(tokenResp.ExpiresIn) * time.Second
	}

	slog.Debug("ChatGPT OAuth code exchange successful")
	return &Token{
		IDToken:      tokenResp.IDToken,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		AccountID:    accountID,
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(expiry),
	}, nil
}

func postForm(ctx context.Context, endpoint string, data url.Values, result any) error {
	return postJSON(ctx, endpoint, "application/x-www-form-urlencoded", bytes.NewBufferString(data.Encode()), result)
}

func postJSON(ctx context.Context, endpoint, contentType string, body io.Reader, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func extractAccountID(idToken, accessToken string) string {
	if id := accountIDFromJWT(idToken); id != "" {
		return id
	}
	return accountIDFromJWT(accessToken)
}

func accountIDFromJWT(token string) string {
	parts := strings.SplitN(token, ".", 4)
	if len(parts) != 3 {
		return ""
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}

	var claims struct {
		ChatGPTAccountID string `json:"chatgpt_account_id"`
		Auth             *struct {
			ChatGPTAccountID string `json:"chatgpt_account_id"`
		} `json:"https://api.openai.com/auth"`
		Organizations []struct {
			ID string `json:"id"`
		} `json:"organizations"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}

	if claims.ChatGPTAccountID != "" {
		return claims.ChatGPTAccountID
	}
	if claims.Auth != nil && claims.Auth.ChatGPTAccountID != "" {
		return claims.Auth.ChatGPTAccountID
	}
	if len(claims.Organizations) > 0 && claims.Organizations[0].ID != "" {
		return claims.Organizations[0].ID
	}
	return ""
}

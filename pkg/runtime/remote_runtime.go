package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/oauth2"

	"github.com/docker/cagent/pkg/api"
	"github.com/docker/cagent/pkg/chat"
	latest "github.com/docker/cagent/pkg/config/v2"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/team"
	"github.com/docker/cagent/pkg/tools"
	"github.com/docker/cagent/pkg/tools/mcp"
)

// RemoteRuntime implements the Interface using a remote client
type RemoteRuntime struct {
	client                  *Client
	currentAgent            string
	agentFilename           string
	sessionID               string
	team                    *team.Team
	pendingOAuthElicitation *ElicitationRequestEvent
}

// RemoteRuntimeOption is a function for configuring the RemoteRuntime
type RemoteRuntimeOption func(*RemoteRuntime)

// WithRemoteCurrentAgent sets the current agent name
func WithRemoteCurrentAgent(agentName string) RemoteRuntimeOption {
	return func(r *RemoteRuntime) {
		r.currentAgent = agentName
	}
}

// WithRemoteAgentFilename sets the agent filename to use with the remote API
func WithRemoteAgentFilename(filename string) RemoteRuntimeOption {
	return func(r *RemoteRuntime) {
		r.agentFilename = filename
	}
}

// NewRemoteRuntime creates a new remote runtime that implements the Interface
func NewRemoteRuntime(client *Client, opts ...RemoteRuntimeOption) (*RemoteRuntime, error) {
	if client == nil {
		return nil, fmt.Errorf("client cannot be nil")
	}

	r := &RemoteRuntime{
		client:        client,
		currentAgent:  "root",
		agentFilename: "agent.yaml", // default
		team:          team.New(),   // empty team, will be populated as needed
	}

	for _, opt := range opts {
		opt(r)
	}

	return r, nil
}

// CurrentAgentName returns the name of the currently active agent
func (r *RemoteRuntime) CurrentAgentName() string {
	return r.currentAgent
}

func (r *RemoteRuntime) CurrentAgentCommands(ctx context.Context) map[string]string {
	return r.readCurrentAgentConfig(ctx).Commands
}

func (r *RemoteRuntime) CurrentWelcomeMessage(ctx context.Context) string {
	return r.readCurrentAgentConfig(ctx).WelcomeMessage
}

// EmitStartupInfo emits initial agent, team, and toolset information for immediate sidebar display
func (r *RemoteRuntime) EmitStartupInfo(ctx context.Context, events chan Event) {
	agentConfig := r.readCurrentAgentConfig(ctx)

	// Emit agent information for sidebar display
	modelID := agentConfig.Model // In v2 config, Model is already a string
	events <- AgentInfo(r.currentAgent, modelID, agentConfig.Description)

	// Emit team information
	availableAgents := r.team.AgentNames()
	events <- TeamInfo(availableAgents, r.currentAgent)

	// For remote runtime, we estimate toolset count from config
	toolsetCount := len(agentConfig.Toolsets)
	events <- ToolsetInfo(toolsetCount, r.currentAgent)
}

func (r *RemoteRuntime) readCurrentAgentConfig(ctx context.Context) latest.AgentConfig {
	cfg, err := r.client.GetAgent(ctx, r.agentFilename)
	if err != nil {
		return latest.AgentConfig{}
	}

	for agentName, agent := range cfg.Agents {
		if agentName == r.currentAgent {
			return agent
		}
	}

	return latest.AgentConfig{}
}

// RunStream starts the agent's interaction loop and returns a channel of events
func (r *RemoteRuntime) RunStream(ctx context.Context, sess *session.Session) <-chan Event {
	slog.Debug("Starting remote runtime stream", "agent", r.currentAgent, "session_id", r.sessionID)
	events := make(chan Event, 128)

	go func() {
		defer close(events)

		// Convert session messages to remote.Message format
		messages := r.convertSessionMessages(sess)

		r.sessionID = sess.ID

		// Start streaming from remote client
		var streamChan <-chan Event
		var err error

		if r.currentAgent != "" && r.currentAgent != "root" {
			streamChan, err = r.client.RunAgentWithAgentName(ctx, r.sessionID, r.agentFilename, r.currentAgent, messages)
		} else {
			streamChan, err = r.client.RunAgent(ctx, r.sessionID, r.agentFilename, messages)
		}

		if err != nil {
			events <- Error(fmt.Sprintf("failed to start remote agent: %v", err))
			return
		}

		for streamEvent := range streamChan {
			if elicitationRequest, ok := streamEvent.(*ElicitationRequestEvent); ok {
				// Store pending OAuth elicitation request
				r.pendingOAuthElicitation = elicitationRequest
			}
			events <- streamEvent
		}
	}()

	return events
}

// Run starts the agent's interaction loop and returns the final messages
func (r *RemoteRuntime) Run(ctx context.Context, sess *session.Session) ([]session.Message, error) {
	eventsChan := r.RunStream(ctx, sess)

	for event := range eventsChan {
		if errEvent, ok := event.(*ErrorEvent); ok {
			return nil, fmt.Errorf("%s", errEvent.Error)
		}
	}

	return sess.GetAllMessages(), nil
}

// Resume allows resuming execution after user confirmation
func (r *RemoteRuntime) Resume(ctx context.Context, confirmationType ResumeType) {
	slog.Debug("Resuming remote runtime", "agent", r.currentAgent, "confirmation_type", confirmationType, "session_id", r.sessionID)

	if r.sessionID == "" {
		slog.Error("Cannot resume: no session ID available")
		return
	}

	if err := r.client.ResumeSession(ctx, r.sessionID, string(confirmationType)); err != nil {
		slog.Error("Failed to resume remote session", "error", err, "session_id", r.sessionID)
	}
}

// Summarize generates a summary for the session
func (r *RemoteRuntime) Summarize(_ context.Context, sess *session.Session, events chan Event) {
	slog.Debug("Summarize not yet implemented for remote runtime", "session_id", r.sessionID)
	// TODO: Implement summarization by either:
	// 1. Adding a summarization endpoint to the remote API
	// 2. Running a summarization agent through the remote client
	events <- SessionSummary(sess.ID, "Summary generation not yet implemented for remote runtime", r.currentAgent)
}

// convertSessionMessages converts session messages to remote API message format
func (r *RemoteRuntime) convertSessionMessages(sess *session.Session) []api.Message {
	sessionMessages := sess.GetAllMessages()
	messages := make([]api.Message, 0, len(sessionMessages))

	for i := range sessionMessages {
		// Only include user and assistant messages for the remote API
		if sessionMessages[i].Message.Role == chat.MessageRoleUser || sessionMessages[i].Message.Role == chat.MessageRoleAssistant {
			messages = append(messages, api.Message{
				Role:    sessionMessages[i].Message.Role,
				Content: sessionMessages[i].Message.Content,
			})
		}
	}

	return messages
}

// ResumeElicitation sends an elicitation response back to a waiting elicitation request
func (r *RemoteRuntime) ResumeElicitation(ctx context.Context, action tools.ElicitationAction, content map[string]any) error {
	slog.Debug("Resuming remote runtime with elicitation response", "agent", r.currentAgent, "action", action, "session_id", r.sessionID)

	err := r.handleOAuthElicitation(ctx, r.pendingOAuthElicitation)
	if err != nil {
		return err
	}
	// TODO: once we get here and the elicitation is the OAuth type, we need to start the managed OAuth flow

	if err := r.client.ResumeElicitation(ctx, r.sessionID, action, content); err != nil {
		return err
	}

	return nil
}

// HandleOAuthElicitation handles OAuth elicitation requests from remote MCP servers
func (r *RemoteRuntime) handleOAuthElicitation(ctx context.Context, req *ElicitationRequestEvent) error {
	slog.Debug("Handling OAuth elicitation request", "server_url", req.Meta["cagent/server_url"])

	// Extract OAuth parameters from metadata
	serverURL, ok := req.Meta["cagent/server_url"].(string)
	if !ok {
		err := fmt.Errorf("server_url missing from elicitation metadata")
		slog.Error("Failed to extract server_url", "error", err)
		_ = r.client.ResumeElicitation(ctx, r.sessionID, "decline", nil)
		return err
	}

	// Extract authorization server metadata
	authServerMetadata, ok := req.Meta["auth_server_metadata"].(map[string]any)
	if !ok {
		err := fmt.Errorf("auth_server_metadata missing from elicitation metadata")
		slog.Error("Failed to extract auth_server_metadata", "error", err)
		_ = r.client.ResumeElicitation(ctx, r.sessionID, "decline", nil)
		return err
	}

	// Unmarshal authorization server metadata
	var authMetadata mcp.AuthorizationServerMetadata
	metadataBytes, err := json.Marshal(authServerMetadata)
	if err != nil {
		slog.Error("Failed to marshal auth_server_metadata", "error", err)
		_ = r.client.ResumeElicitation(ctx, r.sessionID, "decline", nil)
		return fmt.Errorf("failed to marshal auth_server_metadata: %w", err)
	}
	if err := json.Unmarshal(metadataBytes, &authMetadata); err != nil {
		slog.Error("Failed to unmarshal auth_server_metadata", "error", err)
		_ = r.client.ResumeElicitation(ctx, r.sessionID, "decline", nil)
		return fmt.Errorf("failed to unmarshal auth_server_metadata: %w", err)
	}

	slog.Debug("Authorization server metadata extracted", "issuer", authMetadata.Issuer)

	// Create timeout context for OAuth flow (5 minutes)
	oauthCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Create and start callback server
	slog.Debug("Creating OAuth callback server")
	callbackServer, err := mcp.NewCallbackServer()
	if err != nil {
		slog.Error("Failed to create callback server", "error", err)
		_ = r.client.ResumeElicitation(ctx, r.sessionID, "decline", nil)
		return fmt.Errorf("failed to create callback server: %w", err)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := callbackServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("Failed to shutdown callback server", "error", err)
		}
	}()

	if err := callbackServer.Start(); err != nil {
		slog.Error("Failed to start callback server", "error", err)
		_ = r.client.ResumeElicitation(ctx, r.sessionID, "decline", nil)
		return fmt.Errorf("failed to start callback server: %w", err)
	}

	redirectURI := callbackServer.GetRedirectURI()
	slog.Debug("Callback server started", "redirect_uri", redirectURI)

	// Register client
	var clientID, clientSecret string
	if authMetadata.RegistrationEndpoint != "" {
		slog.Debug("Attempting dynamic client registration")
		clientID, clientSecret, err = mcp.RegisterClient(oauthCtx, &authMetadata, redirectURI, nil)
		if err != nil {
			slog.Error("Dynamic client registration failed", "error", err)
			_ = r.client.ResumeElicitation(ctx, r.sessionID, "decline", nil)
			return fmt.Errorf("failed to register client: %w", err)
		}
		slog.Debug("Client registered successfully", "client_id", clientID)
	} else {
		err := fmt.Errorf("authorization server does not support dynamic client registration")
		slog.Error("Client registration not supported", "error", err)
		_ = r.client.ResumeElicitation(ctx, r.sessionID, "decline", nil)
		return err
	}

	// Generate state and PKCE verifier
	state, err := mcp.GenerateState()
	if err != nil {
		slog.Error("Failed to generate state", "error", err)
		_ = r.client.ResumeElicitation(ctx, r.sessionID, "decline", nil)
		return fmt.Errorf("failed to generate state: %w", err)
	}

	callbackServer.SetExpectedState(state)
	verifier := mcp.GeneratePKCEVerifier()

	// Build authorization URL
	authURL := mcp.BuildAuthorizationURL(
		authMetadata.AuthorizationEndpoint,
		clientID,
		redirectURI,
		state,
		oauth2.S256ChallengeFromVerifier(verifier),
		serverURL,
	)

	slog.Debug("Authorization URL built", "url", authURL)

	// Request authorization code (this opens the browser)
	slog.Debug("Requesting authorization code")
	code, receivedState, err := mcp.RequestAuthorizationCode(oauthCtx, authURL, callbackServer, state)
	if err != nil {
		slog.Error("Failed to get authorization code", "error", err)
		_ = r.client.ResumeElicitation(ctx, r.sessionID, "decline", nil)
		return fmt.Errorf("failed to get authorization code: %w", err)
	}

	if receivedState != state {
		err := fmt.Errorf("state mismatch: expected %s, got %s", state, receivedState)
		slog.Error("State mismatch in authorization response", "error", err)
		_ = r.client.ResumeElicitation(ctx, r.sessionID, "decline", nil)
		return err
	}

	slog.Debug("Authorization code received, exchanging for token")

	// Exchange code for token
	token, err := mcp.ExchangeCodeForToken(
		oauthCtx,
		authMetadata.TokenEndpoint,
		code,
		verifier,
		clientID,
		clientSecret,
		redirectURI,
	)
	if err != nil {
		slog.Error("Failed to exchange code for token", "error", err)
		_ = r.client.ResumeElicitation(ctx, r.sessionID, "decline", nil)
		return fmt.Errorf("failed to exchange code for token: %w", err)
	}

	slog.Debug("Token obtained successfully", "token_type", token.TokenType)

	// Send token back to server via ResumeElicitation
	tokenData := map[string]any{
		"access_token": token.AccessToken,
		"token_type":   token.TokenType,
	}
	if token.ExpiresIn > 0 {
		tokenData["expires_in"] = token.ExpiresIn
	}
	if token.RefreshToken != "" {
		tokenData["refresh_token"] = token.RefreshToken
	}

	slog.Debug("Sending token to server")
	if err := r.client.ResumeElicitation(ctx, r.sessionID, tools.ElicitationActionAccept, tokenData); err != nil {
		slog.Error("Failed to send token to server", "error", err)
		return fmt.Errorf("failed to send token to server: %w", err)
	}

	slog.Debug("OAuth flow completed successfully")
	return nil
}

// Verify that RemoteRuntime implements the Interface
var _ Runtime = (*RemoteRuntime)(nil)

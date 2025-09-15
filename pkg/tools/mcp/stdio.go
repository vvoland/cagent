package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/mark3labs/mcp-go/mcp"
)

// NewStdioClient creates a new MCP client that can start an stdio MCP server
func NewStdioClient(command string, args, env []string) *Client {
	slog.Debug("Creating stdio MCP client", "command", command, "args", args)

	c := newStdioCmdClient(command, env, args)

	slog.Debug("Created stdio MCP client successfully")
	return &Client{
		client:  c,
		logType: "command",
		logId:   command,
	}
}

type stdioMCPClient struct {
	command string
	env     []string
	args    []string

	stdin       io.WriteCloser
	requestID   atomic.Int64
	responses   sync.Map
	close       func() error
	initialized atomic.Bool
}

type RPCResponse struct {
	Error    *string
	Response *json.RawMessage
}

type BaseMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func newStdioCmdClient(command string, env, args []string) *stdioMCPClient {
	return &stdioMCPClient{
		command: command,
		env:     env,
		args:    args,
	}
}

func (c *stdioMCPClient) Start(context.Context) error {
	return nil
}

func (c *stdioMCPClient) Initialize(ctx context.Context, request mcp.InitializeRequest) (*mcp.InitializeResult, error) { //nolint:gocritic
	if c.initialized.Load() {
		return nil, errors.New("client already initialized")
	}

	ctxCmd, cancel := context.WithCancel(context.WithoutCancel(ctx))
	cmd := exec.CommandContext(ctxCmd, c.command, c.args...)
	cmd.Env = c.env
	cmd.Cancel = func() error {
		if runtime.GOOS == "windows" {
			return cmd.Process.Kill()
		}
		return cmd.Process.Signal(syscall.SIGTERM)
	}

	var stderr bytes.Buffer
	if slog.Default().Handler().Enabled(ctx, slog.LevelDebug) {
		logLogger := slog.NewLogLogger(slog.Default().Handler(), slog.LevelDebug)
		cmd.Stderr = io.MultiWriter(&stderr, logLogger.Writer())
	} else {
		cmd.Stderr = io.MultiWriter(&stderr)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	c.stdin = stdin

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	c.close = func() error {
		cancel()
		return nil
	}
	go func() {
		_ = cmd.Wait()
		cancel()
	}()
	go func() {
		_ = c.readResponses(bufio.NewReader(stdout))
	}()

	var result mcp.InitializeResult
	errCh := make(chan error)
	go func() {
		errCh <- func() error {
			params := struct {
				ProtocolVersion string                 `json:"protocolVersion"`
				ClientInfo      mcp.Implementation     `json:"clientInfo"`
				Capabilities    mcp.ClientCapabilities `json:"capabilities"`
			}{
				ProtocolVersion: request.Params.ProtocolVersion,
				ClientInfo:      request.Params.ClientInfo,
				Capabilities:    request.Params.Capabilities,
			}

			if err := c.request(ctxCmd, "initialize", params, &result); err != nil {
				return err
			}

			encoder := json.NewEncoder(stdin)
			if err := encoder.Encode(mcp.JSONRPCNotification{
				JSONRPC: mcp.JSONRPC_VERSION,
				Notification: mcp.Notification{
					Method: "notifications/initialized",
				},
			}); err != nil {
				return fmt.Errorf("failed to marshal initialized notification: %w", err)
			}

			c.initialized.Store(true)
			return nil
		}()
	}()

	select {
	case <-ctxCmd.Done():
		return nil, errors.New(stderr.String())
	case <-ctx.Done():
		cancel() // need to also cancel command if timed out or cancelled from parent
		return nil, ctx.Err()
	case err := <-errCh:
		if err != nil {
			return nil, err
		}
	}

	return &result, nil
}

func (c *stdioMCPClient) Close() error {
	err := c.close()

	c.initialized.Store(false)

	c.responses.Range(func(key, value any) bool {
		if ch, ok := value.(chan RPCResponse); ok {
			msg := "client closed"
			select {
			case ch <- RPCResponse{Error: &msg}:
			default:
			}
		}
		c.responses.Delete(key)
		return true
	})

	return err
}

func (c *stdioMCPClient) readResponses(stdout *bufio.Reader) error {
	for {
		buf, err := stdout.ReadBytes('\n')
		if err != nil {
			return err
		}

		var baseMessage BaseMessage
		if err := json.Unmarshal(buf, &baseMessage); err != nil {
			continue
		}

		if baseMessage.ID == nil {
			continue
		}

		if ch, ok := c.responses.LoadAndDelete(*baseMessage.ID); ok {
			responseChan := ch.(chan RPCResponse)

			if baseMessage.Error != nil {
				responseChan <- RPCResponse{
					Error: &baseMessage.Error.Message,
				}
			} else {
				responseChan <- RPCResponse{
					Response: &baseMessage.Result,
				}
			}
		}
	}
}

func (c *stdioMCPClient) sendRequest(ctx context.Context, method string, params any) (*json.RawMessage, error) {
	id := c.requestID.Add(1)
	responseChan := make(chan RPCResponse, 1)
	c.responses.Store(id, responseChan)

	encoder := json.NewEncoder(c.stdin)
	if err := encoder.Encode(mcp.JSONRPCRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      mcp.NewRequestId(id),
		Request: mcp.Request{
			Method: method,
		},
		Params: params,
	}); err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	select {
	case <-ctx.Done():
		// Best-effort cleanup of pending response to avoid leaks when a request is canceled
		c.responses.Delete(id)
		return nil, ctx.Err()
	case response := <-responseChan:
		if response.Error != nil {
			return nil, errors.New(*response.Error)
		}
		return response.Response, nil
	}
}

func (c *stdioMCPClient) request(ctx context.Context, method string, params, v any) error {
	response, err := c.sendRequest(ctx, method, params)
	if err != nil {
		return err
	}

	return json.Unmarshal(*response, &v)
}

func (c *stdioMCPClient) ListTools(ctx context.Context, request mcp.ListToolsRequest) (*mcp.ListToolsResult, error) {
	var result mcp.ListToolsResult
	if err := c.request(ctx, "tools/list", request, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *stdioMCPClient) ListPrompts(ctx context.Context, request mcp.ListPromptsRequest) (*mcp.ListPromptsResult, error) {
	var result mcp.ListPromptsResult
	if err := c.request(ctx, "prompts/list", request, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *stdioMCPClient) ListResources(ctx context.Context, request mcp.ListResourcesRequest) (*mcp.ListResourcesResult, error) {
	var result mcp.ListResourcesResult
	if err := c.request(ctx, "resources/list", request, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *stdioMCPClient) ListResourceTemplates(ctx context.Context, request mcp.ListResourceTemplatesRequest) (*mcp.ListResourceTemplatesResult, error) {
	var result mcp.ListResourceTemplatesResult
	if err := c.request(ctx, "resources/templates/list", request, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *stdioMCPClient) CallTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	response, err := c.sendRequest(ctx, "tools/call", request.Params)
	if err != nil {
		return nil, err
	}

	return mcp.ParseCallToolResult(response)
}

func (c *stdioMCPClient) GetPrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	response, err := c.sendRequest(ctx, "prompts/get", request.Params)
	if err != nil {
		return nil, err
	}

	return mcp.ParseGetPromptResult(response)
}

func (c *stdioMCPClient) ReadResource(ctx context.Context, request mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	response, err := c.sendRequest(ctx, "resources/read", request.Params)
	if err != nil {
		return nil, err
	}

	return mcp.ParseReadResourceResult(response)
}

// Ping checks if the server is alive
func (c *stdioMCPClient) Ping(ctx context.Context) error {
	_, err := c.sendRequest(ctx, "ping", nil)
	return err
}

// Subscribe requests notifications for changes to a specific resource
func (c *stdioMCPClient) Subscribe(ctx context.Context, request mcp.SubscribeRequest) error {
	_, err := c.sendRequest(ctx, "resources/subscribe", request.Params)
	return err
}

// Unsubscribe cancels notifications for a specific resource
func (c *stdioMCPClient) Unsubscribe(ctx context.Context, request mcp.UnsubscribeRequest) error {
	_, err := c.sendRequest(ctx, "resources/unsubscribe", request.Params)
	return err
}

// SetLevel sets the logging level for the server
func (c *stdioMCPClient) SetLevel(ctx context.Context, request mcp.SetLevelRequest) error {
	_, err := c.sendRequest(ctx, "logging/setLevel", request.Params)
	return err
}

// Complete requests completion options for a given argument
func (c *stdioMCPClient) Complete(ctx context.Context, request mcp.CompleteRequest) (*mcp.CompleteResult, error) { //nolint:gocritic // interface requires value type
	response, err := c.sendRequest(ctx, "completion/complete", request.Params)
	if err != nil {
		return nil, err
	}

	var result mcp.CompleteResult
	if err := json.Unmarshal(*response, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListResourcesByPage manually list resources by page
func (c *stdioMCPClient) ListResourcesByPage(ctx context.Context, request mcp.ListResourcesRequest) (*mcp.ListResourcesResult, error) {
	return c.ListResources(ctx, request)
}

// ListResourceTemplatesByPage manually list resource templates by page
func (c *stdioMCPClient) ListResourceTemplatesByPage(ctx context.Context, request mcp.ListResourceTemplatesRequest) (*mcp.ListResourceTemplatesResult, error) {
	return c.ListResourceTemplates(ctx, request)
}

// ListPromptsByPage manually list prompts by page
func (c *stdioMCPClient) ListPromptsByPage(ctx context.Context, request mcp.ListPromptsRequest) (*mcp.ListPromptsResult, error) {
	return c.ListPrompts(ctx, request)
}

// ListToolsByPage manually list tools by page
func (c *stdioMCPClient) ListToolsByPage(ctx context.Context, request mcp.ListToolsRequest) (*mcp.ListToolsResult, error) {
	return c.ListTools(ctx, request)
}

// OnNotification registers a handler for notifications (no-op for stdio client)
func (c *stdioMCPClient) OnNotification(handler func(notification mcp.JSONRPCNotification)) {
	// TODO: Implement notification handling for stdio client if needed
}

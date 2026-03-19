package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/openai/openai-go/v3/responses"
)

const (
	// wsMaxConnectionAge is the maximum lifetime of a WebSocket connection.
	// OpenAI enforces a 60-minute limit; we reconnect slightly earlier.
	wsMaxConnectionAge = 55 * time.Minute
)

// wsConnection holds a WebSocket connection together with bookkeeping
// metadata for the connection pool.
type wsConnection struct {
	conn      *websocket.Conn
	createdAt time.Time

	// lastResponseID is the ID of the most recent response completed on
	// this connection. It can be passed as previous_response_id in subsequent
	// requests to enable server-side context caching.
	lastResponseID string
}

// isExpired returns true when the connection has been open longer than
// wsMaxConnectionAge.
func (c *wsConnection) isExpired() bool {
	return time.Since(c.createdAt) >= wsMaxConnectionAge
}

// wsPool manages a single reusable WebSocket connection to the OpenAI
// Responses API. It is safe for concurrent use; however, because the
// OpenAI WebSocket protocol is sequential (one response at a time),
// callers must not overlap requests on the same pool.
type wsPool struct {
	mu   sync.Mutex
	conn *wsConnection

	// wsURL is the WebSocket endpoint (e.g. wss://api.openai.com/v1/responses).
	wsURL string

	// headerFn returns the HTTP headers (including Authorization) for
	// the WebSocket handshake. It is called each time a new connection
	// is established so that short-lived tokens are refreshed.
	headerFn func(ctx context.Context) (http.Header, error)
}

// newWSPool creates a pool for the given WebSocket URL.
func newWSPool(wsURL string, headerFn func(ctx context.Context) (http.Header, error)) *wsPool {
	return &wsPool{
		wsURL:    wsURL,
		headerFn: headerFn,
	}
}

// Stream opens (or reuses) a WebSocket connection, sends a response.create
// message, and returns a responseEventStream that yields server events.
func (p *wsPool) Stream(
	ctx context.Context,
	params responses.ResponseNewParams,
) (responseEventStream, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Close stale connections.
	if p.conn != nil && p.conn.isExpired() {
		slog.Debug("Closing expired WebSocket connection",
			"age", time.Since(p.conn.createdAt))
		_ = p.conn.conn.Close()
		p.conn = nil
	}

	// Establish a new connection if needed.
	if p.conn == nil {
		headers, err := p.headerFn(ctx)
		if err != nil {
			return nil, fmt.Errorf("websocket pool: headers: %w", err)
		}

		stream, err := dialWebSocket(ctx, p.wsURL, headers, params)
		if err != nil {
			return nil, err
		}

		p.conn = &wsConnection{
			conn:      stream.conn,
			createdAt: time.Now(),
		}

		return &pooledStream{pool: p, inner: stream}, nil
	}

	// Reuse existing connection: send a new response.create.
	stream, err := sendOnExisting(p.conn.conn, params)
	if err != nil {
		// Connection is broken; tear down and retry with a fresh one.
		slog.Warn("Existing WebSocket connection failed, reconnecting", "error", err)
		_ = p.conn.conn.Close()
		p.conn = nil

		headers, err2 := p.headerFn(ctx)
		if err2 != nil {
			return nil, fmt.Errorf("websocket pool: headers on reconnect: %w", err2)
		}
		stream, err2 = dialWebSocket(ctx, p.wsURL, headers, params)
		if err2 != nil {
			return nil, fmt.Errorf("websocket pool: reconnect: %w", err2)
		}
		p.conn = &wsConnection{
			conn:      stream.conn,
			createdAt: time.Now(),
		}
		return &pooledStream{pool: p, inner: stream}, nil
	}

	return &pooledStream{pool: p, inner: stream}, nil
}

// Close shuts down the pooled connection.
func (p *wsPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn != nil {
		_ = p.conn.conn.Close()
		p.conn = nil
	}
}

// sendOnExisting sends a response.create on an already-open connection and
// returns a wsStream that reads events from it.
func sendOnExisting(conn *websocket.Conn, params responses.ResponseNewParams) (*wsStream, error) {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("websocket: marshal params: %w", err)
	}

	msg := wsCreateMessage{
		Type:   "response.create",
		Params: paramsJSON,
	}

	if err := conn.WriteJSON(msg); err != nil {
		return nil, fmt.Errorf("websocket: write response.create: %w", err)
	}

	slog.Debug("WebSocket response.create sent (reused connection)")

	return &wsStream{conn: conn}, nil
}

// pooledStream wraps a wsStream and updates pool state when the response
// finishes. Its Close does NOT close the underlying WebSocket connection
// (which is owned by the pool).
type pooledStream struct {
	pool  *wsPool
	inner *wsStream
}

var _ responseEventStream = (*pooledStream)(nil)

func (s *pooledStream) Next() bool {
	ok := s.inner.Next()
	if !ok {
		return false
	}

	// Track response ID from terminal events for future continuation.
	event := s.inner.Current()
	if isTerminalEvent(event.Type) && event.Response.ID != "" {
		s.pool.mu.Lock()
		if s.pool.conn != nil {
			s.pool.conn.lastResponseID = event.Response.ID
		}
		s.pool.mu.Unlock()
	}

	return true
}

func (s *pooledStream) Current() responses.ResponseStreamEventUnion {
	return s.inner.Current()
}

func (s *pooledStream) Err() error {
	return s.inner.Err()
}

// Close releases the stream but keeps the underlying connection alive in
// the pool for reuse.
func (s *pooledStream) Close() error {
	s.inner.done = true
	// Do NOT close the WebSocket connection—it stays in the pool.
	return nil
}

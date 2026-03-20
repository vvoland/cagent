package remote

import (
	"context"
	"net"
	"net/http"
	"net/url"

	"github.com/docker/docker-agent/pkg/desktop"
	socket "github.com/docker/docker-agent/pkg/desktop/socket"
)

// NewTransport returns an HTTP transport that uses Docker Desktop proxy if available.
func NewTransport(ctx context.Context) http.RoundTripper {
	t, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return http.DefaultTransport
	}
	transport := t.Clone()

	if desktop.IsDockerDesktopRunning(ctx) {
		// Route all traffic through Docker Desktop's HTTP proxy socket
		// Set a dummy proxy URL - the actual connection happens via DialContext
		transport.Proxy = http.ProxyURL(&url.URL{
			Scheme: "http",
		})
		// Override the dialer to connect to the Unix socket for the proxy
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return socket.DialUnix(ctx, desktop.Paths().ProxySocket)
		}
	}

	return transport
}

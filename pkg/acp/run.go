package acp

import (
	"context"
	"io"
	"log/slog"

	acpsdk "github.com/coder/acp-go-sdk"

	"github.com/docker/cagent/pkg/config"
)

func Run(ctx context.Context, agentFilename string, stdin io.Reader, stdout io.Writer, runConfig *config.RuntimeConfig) error {
	slog.Debug("Starting ACP server", "agent", agentFilename)

	agentSource, err := config.Resolve(agentFilename)
	if err != nil {
		return err
	}

	acpAgent := NewAgent(agentSource, runConfig)
	conn := acpsdk.NewAgentSideConnection(acpAgent, stdout, stdin)
	conn.SetLogger(slog.Default())
	acpAgent.SetAgentConnection(conn)
	defer acpAgent.Stop(ctx)

	slog.Debug("acp started, waiting for conn")
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-conn.Done():
		return nil
	}
}

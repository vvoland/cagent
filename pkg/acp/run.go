package acp

import (
	"context"
	"io"
	"log/slog"

	acpsdk "github.com/coder/acp-go-sdk"

	"github.com/docker/cagent/pkg/agentfile"
	"github.com/docker/cagent/pkg/config"
)

type discardOutput struct{}

func (d *discardOutput) Printf(string, ...any) {}

func Run(ctx context.Context, agentFilename string, stdin io.Reader, stdout io.Writer, runConfig *config.RuntimeConfig) error {
	agentFilename, err := agentfile.Resolve(ctx, &discardOutput{}, agentFilename)
	if err != nil {
		return err
	}

	slog.Debug("Starting ACP server", "agent_file", agentFilename)

	acpAgent := NewAgent(agentFilename, runConfig)
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

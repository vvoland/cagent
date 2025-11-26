package a2a

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/server/adka2a"
	"google.golang.org/adk/session"

	"github.com/docker/cagent/pkg/agentfile"
	"github.com/docker/cagent/pkg/cli"
	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/teamloader"
	"github.com/docker/cagent/pkg/version"
)

func Run(ctx context.Context, out *cli.Printer, agentFilename, agentName string, runConfig *config.RuntimeConfig, ln net.Listener) error {
	slog.Debug("Starting A2A server", "agent", agentName, "addr", ln.Addr().String())

	agentFilename, err := agentfile.Resolve(ctx, out, agentFilename)
	if err != nil {
		return err
	}

	t, err := teamloader.Load(ctx, agentFilename, runConfig)
	if err != nil {
		return fmt.Errorf("failed to load agents: %w", err)
	}
	defer func() {
		if err := t.StopToolSets(ctx); err != nil {
			slog.Error("Failed to stop tool sets", "error", err)
		}
	}()

	adkAgent, err := newCAgentAdapter(t, agentName)
	if err != nil {
		return fmt.Errorf("failed to create ADK agent adapter: %w", err)
	}

	baseURL := &url.URL{Scheme: "http", Host: ln.Addr().String()}

	out.Println("A2A server listening on", baseURL.String())

	agentPath := "/invoke"
	agentCard := &a2a.AgentCard{
		Name:               adkAgent.Name(),
		Description:        adkAgent.Description(),
		Skills:             adka2a.BuildAgentSkills(adkAgent),
		PreferredTransport: a2a.TransportProtocolJSONRPC,
		URL:                baseURL.JoinPath(agentPath).String(),
		Capabilities:       a2a.AgentCapabilities{Streaming: true},
		Version:            version.Version,
		DefaultInputModes:  []string{},
		DefaultOutputModes: []string{},
	}

	executor := newExecutorWrapper(adka2a.ExecutorConfig{
		RunnerConfig: runner.Config{
			AppName:        adkAgent.Name(),
			Agent:          adkAgent,
			SessionService: session.InMemoryService(),
		},
	})

	// Start server
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodPost, http.MethodOptions},
		AllowHeaders: []string{"Content-Type", "Accept"},
		MaxAge:       86400,
	}))
	e.Use(middleware.Logger())

	e.GET(a2asrv.WellKnownAgentCardPath, echo.WrapHandler(a2asrv.NewStaticAgentCardHandler(agentCard)))
	e.POST(agentPath, echo.WrapHandler(a2asrv.NewJSONRPCHandler(a2asrv.NewHandler(executor))))

	if err := e.Server.Serve(ln); err != nil && ctx.Err() == nil {
		slog.Error("Failed to start server", "error", err)
		return err
	}

	return nil
}

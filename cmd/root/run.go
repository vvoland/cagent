package root

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"

	"github.com/docker/cagent/pkg/aliases"
	"github.com/docker/cagent/pkg/app"
	"github.com/docker/cagent/pkg/cli"
	"github.com/docker/cagent/pkg/content"
	"github.com/docker/cagent/pkg/remote"
	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/team"
	"github.com/docker/cagent/pkg/teamloader"
	"github.com/docker/cagent/pkg/telemetry"
	"github.com/docker/cagent/pkg/tui"
)

var (
	agentsDir      string
	autoApprove    bool
	attachmentPath string
	workingDir     string
	useTUI         bool
	remoteAddress  string
	dryRun         bool
	modelOverrides []string
)

// NewRunCmd creates a new run command
func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <agent-name> [message|-]",
		Short: "Run an agent",
		Long:  `Run an agent with the specified configuration and prompt`,
		Example: `  cagent run ./agent.yaml
  cagent run ./team.yaml --agent root
  cagent run ./echo.yaml "INSTRUCTIONS"
  echo "INSTRUCTIONS" | cagent run ./echo.yaml -`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd.Context(), args)
		},
	}

	cmd.PersistentFlags().StringVarP(&agentName, "agent", "a", "root", "Name of the agent to run")
	cmd.PersistentFlags().StringVar(&workingDir, "working-dir", "", "Set the working directory for the session (applies to tools and relative paths)")
	cmd.PersistentFlags().BoolVar(&autoApprove, "yolo", false, "Automatically approve all tool calls without prompting")
	cmd.PersistentFlags().StringVar(&attachmentPath, "attach", "", "Attach an image file to the message")
	cmd.PersistentFlags().BoolVar(&useTUI, "tui", true, "Run the agent with a Terminal User Interface (TUI)")
	cmd.PersistentFlags().StringVar(&remoteAddress, "remote", "", "Use remote runtime with specified address (only supported with TUI)")
	cmd.PersistentFlags().StringArrayVar(&modelOverrides, "model", nil, "Override agent model: [agent=]provider/model (repeatable)")
	addGatewayFlags(cmd)
	addRuntimeConfigFlags(cmd)

	return cmd
}

func runCommand(ctx context.Context, args []string) error {
	telemetry.TrackCommand("run", args)
	setupOtel(ctx)
	return doRunCommand(ctx, args, false)
}

func setupOtel(ctx context.Context) {
	if enableOtel {
		if err := initOTelSDK(ctx); err != nil {
			slog.Warn("Failed to initialize OpenTelemetry SDK", "error", err)
		} else {
			slog.Debug("OpenTelemetry SDK initialized successfully")
		}
	}
}

func doRunCommand(ctx context.Context, args []string, exec bool) error {
	slog.Debug("Starting agent", "agent", agentName, "debug_mode", debugMode)

	if err := validateRemoteFlag(exec); err != nil {
		return err
	}

	if err := setupWorkingDirectory(); err != nil {
		return err
	}

	agentFileName, err := resolveAgentFile(ctx, args[0])
	if err != nil {
		return err
	}

	t, err := loadAgents(ctx, agentFileName)
	if err != nil {
		return err
	}

	var rt runtime.Runtime
	var sess *session.Session
	if remoteAddress != "" {
		rt, sess, err = createRemoteRuntimeAndSession(ctx, args[0])
		if err != nil {
			return err
		}
	} else {
		rt, sess, err = createLocalRuntimeAndSession(t)
		if err != nil {
			return err
		}
	}

	if exec {
		return handleExecMode(ctx, agentFileName, rt, sess, args)
	}

	if !useTUI {
		return handleCLIMode(ctx, agentFileName, rt, sess, args)
	}

	return handleTUIMode(ctx, agentFileName, rt, t, sess, args)
}

func setupWorkingDirectory() error {
	if workingDir == "" {
		return nil
	}

	absWd, err := filepath.Abs(workingDir)
	if err != nil {
		return fmt.Errorf("invalid working directory: %w", err)
	}

	info, err := os.Stat(absWd)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("working directory does not exist or is not a directory: %s", absWd)
	}

	if err := os.Chdir(absWd); err != nil {
		return fmt.Errorf("failed to change working directory: %w", err)
	}

	_ = os.Setenv("PWD", absWd)
	slog.Debug("Working directory set", "dir", absWd)
	return nil
}

// resolveAgentFile resolves an agent file reference (local file or OCI image) to a local file path
func resolveAgentFile(ctx context.Context, agentFilename string) (string, error) {
	if remoteAddress != "" {
		return agentFilename, nil
	}
	// Try to resolve as an alias first
	if aliasStore, err := aliases.Load(); err == nil {
		if resolvedPath, ok := aliasStore.Get(agentFilename); ok {
			slog.Debug("Resolved alias", "alias", agentFilename, "path", resolvedPath)
			agentFilename = resolvedPath
		}
	}

	ext := strings.ToLower(filepath.Ext(agentFilename))
	if ext == ".yaml" || ext == ".yml" || strings.HasPrefix(agentFilename, "/dev/fd/") {
		// Treat as local YAML file: resolve to absolute path so later chdir doesn't break it
		// TODO(rumpl): Why are we checking for newlines here?
		if !strings.Contains(agentFilename, "\n") {
			if abs, err := filepath.Abs(agentFilename); err == nil {
				agentFilename = abs
			}
		}
		if !fileExists(agentFilename) {
			return "", fmt.Errorf("agent file not found: %s", agentFilename)
		}
		return agentFilename, nil
	}

	// Treat as an OCI image reference. Try local store first, otherwise pull then load.
	a, err := fromStore(agentFilename)
	if err != nil {
		fmt.Println("Pulling agent", agentFilename)
		if _, pullErr := remote.Pull(agentFilename); pullErr != nil {
			return "", fmt.Errorf("failed to pull OCI image %s: %w", agentFilename, pullErr)
		}
		// Retry after pull
		a, err = fromStore(agentFilename)
		if err != nil {
			return "", fmt.Errorf("failed to load agent from store after pull: %w", err)
		}
	}

	// Write the fetched content to a temporary YAML file
	tmpFile, err := os.CreateTemp("", "agentfile-*.yaml")
	if err != nil {
		return "", err
	}
	tmpFilename := tmpFile.Name()
	if _, err := tmpFile.WriteString(a); err != nil {
		tmpFile.Close()
		os.Remove(tmpFilename)
		return "", err
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFilename)
		return "", err
	}
	go func() {
		<-ctx.Done()
		os.Remove(tmpFilename)
	}()
	return tmpFilename, nil
}

func loadAgents(ctx context.Context, agentFilename string) (*team.Team, error) {
	if runConfig.RedirectURI == "" {
		runConfig.RedirectURI = "http://localhost:8083/oauth-callback"
	}

	t, err := teamloader.Load(ctx, agentFilename, runConfig, teamloader.WithModelOverrides(modelOverrides))
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		if err := t.StopToolSets(ctx); err != nil {
			slog.Error("Failed to stop tool sets", "error", err)
		}
	}()

	return t, nil
}

func validateRemoteFlag(exec bool) error {
	if remoteAddress != "" && (!useTUI || exec) {
		return fmt.Errorf("--remote flag can only be used with TUI mode")
	}
	return nil
}

func createRemoteRuntimeAndSession(ctx context.Context, originalFilename string) (runtime.Runtime, *session.Session, error) {
	remoteClient, err := runtime.NewClient(remoteAddress)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create remote client: %w", err)
	}

	sessTemplate := session.New()
	sessTemplate.ToolsApproved = autoApprove
	sess, err := remoteClient.CreateSession(ctx, sessTemplate)
	if err != nil {
		return nil, nil, err
	}

	remoteRt, err := runtime.NewRemoteRuntime(remoteClient,
		runtime.WithRemoteCurrentAgent(agentName),
		runtime.WithRemoteAgentFilename(originalFilename),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create remote runtime: %w", err)
	}

	slog.Debug("Using remote runtime", "address", remoteAddress, "agent", agentName)
	return remoteRt, sess, nil
}

func createLocalRuntimeAndSession(t *team.Team) (runtime.Runtime, *session.Session, error) {
	agent, err := t.Agent(agentName)
	if err != nil {
		return nil, nil, err
	}

	sess := session.New(session.WithMaxIterations(agent.MaxIterations()))
	sess.ToolsApproved = autoApprove

	tracer := otel.Tracer(AppName)

	localRt, err := runtime.New(t,
		runtime.WithCurrentAgent(agentName),
		runtime.WithTracer(tracer),
		runtime.WithRootSessionID(sess.ID),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create runtime: %w", err)
	}

	slog.Debug("Using local runtime", "agent", agentName)
	return localRt, sess, nil
}

func handleExecMode(ctx context.Context, agentFilename string, rt runtime.Runtime, sess *session.Session, args []string) error {
	execArgs := []string{"exec"}
	if len(args) == 2 {
		execArgs = append(execArgs, args[1])
	} else {
		execArgs = append(execArgs, "Follow the default instructions")
	}

	if dryRun {
		fmt.Println("Dry run mode enabled. Agent initialized but will not execute.")
		return nil
	}

	err := cli.Run(ctx, cli.Config{
		AppName:        AppName,
		AttachmentPath: attachmentPath,
	}, agentFilename, rt, sess, execArgs)
	if cliErr, ok := err.(cli.RuntimeError); ok {
		return RuntimeError{Err: cliErr.Err}
	}
	return err
}

func handleCLIMode(ctx context.Context, agentFilename string, rt runtime.Runtime, sess *session.Session, args []string) error {
	err := cli.Run(ctx, cli.Config{
		AppName:        AppName,
		AttachmentPath: attachmentPath,
	}, agentFilename, rt, sess, args)
	if cliErr, ok := err.(cli.RuntimeError); ok {
		return RuntimeError{Err: cliErr.Err}
	}
	return err
}

func readInitialMessage(args []string) (*string, error) {
	if len(args) < 2 {
		return nil, nil
	}

	if args[1] == "-" {
		buf, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read from stdin: %w", err)
		}
		text := string(buf)
		return &text, nil
	}

	return &args[1], nil
}

func handleTUIMode(ctx context.Context, agentFilename string, rt runtime.Runtime, t *team.Team, sess *session.Session, args []string) error {
	firstMessage, err := readInitialMessage(args)
	if err != nil {
		return err
	}

	a := app.New("cagent", agentFilename, rt, t, sess, firstMessage)
	m := tui.New(a)

	progOpts := []tea.ProgramOption{
		tea.WithAltScreen(),
		tea.WithContext(ctx),
		tea.WithFilter(tui.MouseEventFilter),
		tea.WithMouseCellMotion(),
		tea.WithMouseAllMotion(),
	}

	p := tea.NewProgram(m, progOpts...)

	go a.Subscribe(ctx, p)

	_, err = p.Run()
	return err
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	exists := err == nil
	return exists
}

func fromStore(reference string) (string, error) {
	store, err := content.NewStore()
	if err != nil {
		return "", err
	}

	img, err := store.GetArtifactImage(reference)
	if err != nil {
		return "", err
	}

	layers, err := img.Layers()
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	layer := layers[0]
	b, err := layer.Uncompressed()
	if err != nil {
		return "", err
	}

	_, err = io.Copy(&buf, b)
	if err != nil {
		return "", err
	}
	b.Close()

	return buf.String(), nil
}

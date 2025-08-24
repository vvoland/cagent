package root

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"

	"github.com/docker/cagent/pkg/chat"
	"github.com/docker/cagent/pkg/content"
	"github.com/docker/cagent/pkg/evaluation"
	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/teamloader"
)

var autoApprove bool
var attachmentPath string

// initOTelSDK initializes OpenTelemetry SDK with OTLP exporter
func initOTelSDK(ctx context.Context) (shutdown func(context.Context) error, err error) {
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("cagent"),
			semconv.ServiceVersion("dev"), // TODO: use actual version
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	var traceExporter trace.SpanExporter
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	// Only initialize if endpoint is configured
	if endpoint != "" {
		traceExporter, err = otlptracehttp.New(ctx,
			otlptracehttp.WithEndpoint(endpoint),
			otlptracehttp.WithInsecure(), // TODO: make configurable
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create trace exporter: %w", err)
		}
	}

	// Configure tracer provider
	var tracerProviderOpts []trace.TracerProviderOption
	tracerProviderOpts = append(tracerProviderOpts, trace.WithResource(res))

	if traceExporter != nil {
		tracerProviderOpts = append(tracerProviderOpts,
			trace.WithBatcher(traceExporter,
				trace.WithBatchTimeout(5*time.Second),
				trace.WithMaxExportBatchSize(512),
			),
		)
	}

	tp := trace.NewTracerProvider(tracerProviderOpts...)
	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}

var workingDir string

// NewRunCmd creates a new run command
func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <agent-name> [message|-]",
		Short: "Run an agent",
		Long:  `Run an agent with the specified configuration and prompt`,
		Example: `  cagent run ./agent.yaml
  cagent run ./team.yaml --agent root
  cagent run ./echo.yaml "ECHO"
  echo "ECHO" | cagent run ./echo.yaml -`,
		Args: cobra.RangeArgs(1, 2),
		RunE: runAgentCommand,
	}

	cmd.PersistentFlags().StringVarP(&agentName, "agent", "a", "root", "Name of the agent to run")
	cmd.PersistentFlags().BoolVar(&autoApprove, "yolo", false, "Automatically approve all tool calls without prompting")
	cmd.PersistentFlags().StringSliceVar(&runConfig.EnvFiles, "env-from-file", nil, "Set environment variables from file")
	cmd.PersistentFlags().StringVar(&attachmentPath, "attach", "", "Attach an image file to the message")
	cmd.PersistentFlags().StringVar(&workingDir, "working-dir", "", "Set the working directory for the session (applies to tools and relative paths)")
	addGatewayFlags(cmd)

	return cmd
}

func runAgentCommand(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	agentFilename := args[0]

	logger := newLogger()

	logger.Debug("Starting agent", "agent", agentName, "debug_mode", debugMode)

	if enableOtel {
		shutdown, err := initOTelSDK(ctx)
		if err != nil {
			logger.Warn("Failed to initialize OpenTelemetry SDK", "error", err)
		} else if shutdown != nil {
			defer func() {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := shutdown(shutdownCtx); err != nil {
					logger.Warn("Failed to shutdown OpenTelemetry SDK", "error", err)
				}
			}()
			logger.Debug("OpenTelemetry SDK initialized successfully")
		}
	}

	// Resolve agentFilename to an absolute path early so changing cwd won't break path resolution
	if !strings.Contains(agentFilename, "\n") {
		if abs, err := filepath.Abs(agentFilename); err == nil {
			agentFilename = abs
		}
	}

	// If working-dir was provided, validate and change process working directory
	if workingDir != "" {
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
		logger.Debug("Working directory set", "dir", absWd)
	}

	if !fileExists(agentFilename) {
		a, err := fromStore(agentFilename)
		if err != nil {
			return err
		}
		tmpFile, err := os.CreateTemp("", "agentfile-*.yaml")
		if err != nil {
			return err
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.WriteString(a); err != nil {
			tmpFile.Close()
			return err
		}
		if err := tmpFile.Close(); err != nil {
			return err
		}
		agentFilename = tmpFile.Name()
	}

	agents, err := teamloader.Load(ctx, agentFilename, runConfig, logger)
	if err != nil {
		return err
	}
	defer func() {
		if err := agents.StopToolSets(); err != nil {
			logger.Error("Failed to stop tool sets", "error", err)
		}
	}()

	tracer := otel.Tracer("cagent")

	rt := runtime.New(logger, agents,
		runtime.WithCurrentAgent(agentName),
		runtime.WithAutoRunTools(autoApprove),
		runtime.WithTracer(tracer),
	)

	sess := session.New(logger)

	blue := color.New(color.FgBlue).SprintfFunc()
	yellow := color.New(color.FgYellow).SprintfFunc()
	green := color.New(color.FgGreen).SprintfFunc()

	// If the last received event was an error, return it. That way the exit code
	// will be non-zero if the agent failed.
	var lastErr error

	oneLoop := func(text string, scannerConfirmations *bufio.Scanner) error {
		userInput := strings.TrimSpace(text)
		if userInput == "" {
			return nil
		}

		handled, err := runUserCommand(userInput, sess, rt, ctx)
		if err != nil {
			return err
		}

		if handled {
			return nil
		}

		// Parse for /attach commands in the message
		messageText, attachPath := parseAttachCommand(userInput)

		// Use either the per-message attachment or the global one
		finalAttachPath := attachPath
		if finalAttachPath == "" {
			finalAttachPath = attachmentPath
		}

		sess.AddMessage(createUserMessageWithAttachment(agentFilename, messageText, finalAttachPath))

		first := false
		for event := range rt.RunStream(ctx, sess) {
			if !first {
				fmt.Printf("%s", blue("[%s]: ", rt.CurrentAgent().Name()))
				first = true
			}
			lastErr = nil
			switch e := event.(type) {
			case *runtime.AgentChoiceEvent:
				fmt.Printf("%s", e.Choice.Delta.Content)
			case *runtime.ToolCallConfirmationEvent:
				fmt.Printf("%s", yellow("\n%s(%s)\n", e.ToolCall.Function.Name, e.ToolCall.Function.Arguments))
				fmt.Println("\nCan I run this tool? (y/a/n)")
				scannerConfirmations.Scan()
				text := scannerConfirmations.Text()
				switch text {
				case "y":
					rt.Resume(ctx, string(runtime.ResumeTypeApprove))
				case "a":
					sess.ToolsApproved = true
					rt.Resume(ctx, string(runtime.ResumeTypeApproveSession))
				case "n":
					rt.Resume(ctx, string(runtime.ResumeTypeReject))
				}
			case *runtime.ToolCallEvent:
				fmt.Printf("%s", yellow("\n%s(%s)\n", e.ToolCall.Function.Name, e.ToolCall.Function.Arguments))
			case *runtime.ToolCallResponseEvent:
				fmt.Printf("%s", green("done(%s)\n", e.ToolCall.Function.Name))
			case *runtime.ErrorEvent:
				fmt.Printf("%s\n", e.Error)
				lastErr = e.Error
			}
		}
		fmt.Println()
		return nil
	}

	if len(args) == 2 {
		if args[1] == "-" {
			buf, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read from stdin: %w", err)
			}

			if err := oneLoop(string(buf), bufio.NewScanner(os.Stdin)); err != nil {
				return err
			}
		} else {
			if err := oneLoop(args[1], bufio.NewScanner(os.Stdin)); err != nil {
				return err
			}
		}
	} else {
		fmt.Println(blue("\nEnter your messages (Ctrl+C to exit):"))
		scanner := bufio.NewScanner(os.Stdin)
		for {
			fmt.Print(blue("> "))

			if !scanner.Scan() {
				break
			}

			if err := oneLoop(scanner.Text(), scanner); err != nil {
				return err
			}
		}

		if err := scanner.Err(); err != nil {
			return err
		}
	}

	return lastErr
}

func runUserCommand(userInput string, sess *session.Session, rt *runtime.Runtime, ctx context.Context) (bool, error) {
	yellow := color.New(color.FgYellow).SprintfFunc()
	switch userInput {
	case "/exit":
		os.Exit(0)
	case "/eval":
		err := evaluation.Save(sess)
		if err == nil {
			fmt.Printf("%s\n", yellow("Evaluation saved"))
			return true, err
		}
		return true, nil
	case "/usage":
		fmt.Printf("%s\n", yellow("Input tokens: %d", sess.InputTokens))
		fmt.Printf("%s\n", yellow("Output tokens: %d", sess.OutputTokens))
		return true, nil
	case "/reset":
		// Reset session items
		sess.Messages = []session.Item{}
		return true, nil
	case "/compact":
		// Generate a summary of the session and compact the history
		fmt.Printf("%s\n", yellow("Generating summary..."))

		// Create a channel to capture summary events
		events := make(chan runtime.Event, 100)

		// Generate the summary
		rt.Summarize(ctx, sess, events)

		// Process events and show the summary
		close(events)
		summaryGenerated := false
		for event := range events {
			if summaryEvent, ok := event.(*runtime.SessionSummaryEvent); ok {
				fmt.Printf("%s\n", yellow("Summary generated and added to session"))
				fmt.Printf("Summary: %s\n", summaryEvent.Summary)
				summaryGenerated = true
			}
		}

		if !summaryGenerated {
			fmt.Printf("%s\n", yellow("No summary generated"))
		}

		return true, nil
	}

	return false, nil
}

// parseAttachCommand parses user input for /attach commands
// Returns the message text (with /attach commands removed) and the attachment path
func parseAttachCommand(input string) (messageText, attachPath string) {
	lines := strings.Split(input, "\n")
	var messageLines []string

	for _, line := range lines {
		// Look for /attach anywhere in the line
		attachIndex := strings.Index(line, "/attach ")
		if attachIndex != -1 {
			// Extract the part before /attach
			beforeAttach := line[:attachIndex]

			// Extract the part after /attach (starting after "/attach ")
			afterAttachStart := attachIndex + 8 // Length of "/attach "
			if afterAttachStart < len(line) {
				afterAttach := line[afterAttachStart:]

				// Split on spaces to get the file path (first token) and any remaining text
				tokens := strings.Fields(afterAttach)
				if len(tokens) > 0 {
					attachPath = tokens[0]

					// Reconstruct the line with /attach and file path removed
					var remainingText string
					if len(tokens) > 1 {
						remainingText = strings.Join(tokens[1:], " ")
					}

					// Combine the text before /attach and any text after the file path
					var parts []string
					if strings.TrimSpace(beforeAttach) != "" {
						parts = append(parts, strings.TrimSpace(beforeAttach))
					}
					if remainingText != "" {
						parts = append(parts, remainingText)
					}
					reconstructedLine := strings.Join(parts, " ")
					if reconstructedLine != "" {
						messageLines = append(messageLines, reconstructedLine)
					}
				}
			}
		} else {
			// Keep lines without /attach commands
			messageLines = append(messageLines, line)
		}
	}

	// Join the message lines back together
	messageText = strings.TrimSpace(strings.Join(messageLines, "\n"))
	return messageText, attachPath
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

// createUserMessageWithAttachment creates a user message with optional image attachment
func createUserMessageWithAttachment(agentFilename, userContent, attachmentPath string) *session.Message {
	if attachmentPath == "" {
		return session.UserMessage(agentFilename, userContent)
	}

	// Convert file to data URL
	dataURL, err := fileToDataURL(attachmentPath)
	if err != nil {
		fmt.Printf("Warning: Failed to attach file %s: %v\n", attachmentPath, err)
		return session.UserMessage(agentFilename, userContent)
	}

	// Ensure we have some text content when attaching a file
	textContent := userContent
	if strings.TrimSpace(textContent) == "" {
		textContent = "Please analyze this attached file."
	}

	// Create message with multi-content including text and image
	multiContent := []chat.MessagePart{
		{
			Type: chat.MessagePartTypeText,
			Text: textContent,
		},
		{
			Type: chat.MessagePartTypeImageURL,
			ImageURL: &chat.MessageImageURL{
				URL:    dataURL,
				Detail: chat.ImageURLDetailAuto,
			},
		},
	}

	return &session.Message{
		AgentFilename: agentFilename,
		AgentName:     "",
		Message: chat.Message{
			Role:         chat.MessageRoleUser,
			MultiContent: multiContent,
		},
	}
}

// fileToDataURL converts a file to a data URL
func fileToDataURL(filePath string) (string, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", filePath)
	}

	// Read file content
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Determine MIME type based on file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	var mimeType string
	switch ext {
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".png":
		mimeType = "image/png"
	case ".gif":
		mimeType = "image/gif"
	case ".webp":
		mimeType = "image/webp"
	case ".bmp":
		mimeType = "image/bmp"
	case ".svg":
		mimeType = "image/svg+xml"
	default:
		return "", fmt.Errorf("unsupported image format: %s", ext)
	}

	// Encode to base64
	encoded := base64.StdEncoding.EncodeToString(fileBytes)

	// Create data URL
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)

	return dataURL, nil
}

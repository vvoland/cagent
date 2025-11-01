package cli

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/cagent/pkg/chat"
	"github.com/docker/cagent/pkg/evaluation"
	"github.com/docker/cagent/pkg/input"
	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/telemetry"
)

// RuntimeError wraps runtime errors to distinguish them from usage errors
type RuntimeError struct {
	Err error
}

func (e RuntimeError) Error() string {
	return e.Err.Error()
}

func (e RuntimeError) Unwrap() error {
	return e.Err
}

// Config holds configuration for running an agent in CLI mode
type Config struct {
	AppName        string
	AttachmentPath string
}

// Run executes an agent in non-TUI mode, handling user input and runtime events
func Run(ctx context.Context, cfg Config, agentFilename string, rt runtime.Runtime, sess *session.Session, args []string) error {
	// Create a cancellable context for this agentic loop and wire Ctrl+C to cancel it
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Ensure telemetry is initialized and add to context so runtime can access it
	telemetry.EnsureGlobalTelemetryInitialized()
	if telemetryClient := telemetry.GetGlobalTelemetryClient(); telemetryClient != nil {
		ctx = telemetry.WithClient(ctx, telemetryClient)
	}

	sess.Title = "Running agent"
	// If the last received event was an error, return it. That way the exit code
	// will be non-zero if the agent failed.
	var lastErr error

	oneLoop := func(text string, rd io.Reader) error {
		userInput := strings.TrimSpace(text)
		if userInput == "" {
			return nil
		}

		userInput = runtime.ResolveCommand(ctx, rt, userInput)

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
			finalAttachPath = cfg.AttachmentPath
		}

		sess.AddMessage(createUserMessageWithAttachment(agentFilename, messageText, finalAttachPath))

		firstLoop := true
		lastAgent := rt.CurrentAgentName()
		llmIsTyping := false
		reasoningStarted := false // Track if we've printed "Thinking:" prefix
		var lastConfirmedToolCallID string
		for event := range rt.RunStream(ctx, sess) {
			agentName := event.GetAgentName()
			if agentName != "" && (firstLoop || lastAgent != agentName) {
				if !firstLoop {
					if llmIsTyping {
						fmt.Println()
						llmIsTyping = false
					}
					fmt.Println()
				}
				PrintAgentName(agentName)
				firstLoop = false
				lastAgent = agentName
				reasoningStarted = false // Reset reasoning state on agent change
			}
			switch e := event.(type) {
			case *runtime.AgentChoiceEvent:
				agentChanged := lastAgent != e.AgentName
				if !llmIsTyping {
					// Only add newline if we're not already typing
					if !agentChanged {
						fmt.Println()
					}
					llmIsTyping = true
				}
				// Add newline when transitioning from reasoning to regular content
				if reasoningStarted {
					fmt.Println()
				}
				reasoningStarted = false // Reset when regular content starts
				fmt.Printf("%s", e.Content)
			case *runtime.AgentChoiceReasoningEvent:
				if !reasoningStarted {
					// First reasoning chunk: print prefix
					prefix := "Thinking: "
					if e.AgentName != "" && e.AgentName != "root" {
						prefix = prefix + e.AgentName + ": "
					}
					fmt.Printf("\n%s", White(prefix))
					reasoningStarted = true
				}
				// Continue printing reasoning content
				fmt.Printf("%s", White(e.Content))
			case *runtime.ToolCallConfirmationEvent:
				if llmIsTyping {
					fmt.Println()
					llmIsTyping = false
				}
				result := PrintToolCallWithConfirmation(ctx, e.ToolCall, rd)
				// If interrupted, skip resuming; the runtime will notice context cancellation and stop
				if ctx.Err() != nil {
					continue
				}
				lastConfirmedToolCallID = e.ToolCall.ID // Store the ID to avoid duplicate printing
				switch result {
				case ConfirmationApprove:
					rt.Resume(ctx, runtime.ResumeTypeApprove)
				case ConfirmationApproveSession:
					sess.ToolsApproved = true
					rt.Resume(ctx, runtime.ResumeTypeApproveSession)
				case ConfirmationReject:
					rt.Resume(ctx, runtime.ResumeTypeReject)
					lastConfirmedToolCallID = "" // Clear on reject since tool won't execute
				case ConfirmationAbort:
					// Stop the agent loop immediately
					cancel()
					continue
				}
			case *runtime.ToolCallEvent:
				if llmIsTyping {
					fmt.Println()
					llmIsTyping = false
				}
				// Only print if this wasn't already shown during confirmation
				if e.ToolCall.ID != lastConfirmedToolCallID {
					PrintToolCall(e.ToolCall)
				}
			case *runtime.ToolCallResponseEvent:
				if llmIsTyping {
					fmt.Println()
					llmIsTyping = false
				}
				PrintToolCallResponse(e.ToolCall, e.Response)
				// Clear the confirmed ID after the tool completes
				if e.ToolCall.ID == lastConfirmedToolCallID {
					lastConfirmedToolCallID = ""
				}
			case *runtime.ErrorEvent:
				if llmIsTyping {
					fmt.Println()
					llmIsTyping = false
				}
				lowerErr := strings.ToLower(e.Error)
				if strings.Contains(lowerErr, "context cancel") && ctx.Err() != nil { // treat Ctrl+C cancellations as non-errors
					lastErr = nil
				} else {
					lastErr = fmt.Errorf("%s", e.Error)
					PrintError(lastErr)
				}
			case *runtime.MaxIterationsReachedEvent:
				if llmIsTyping {
					fmt.Println()
					llmIsTyping = false
				}

				result := PromptMaxIterationsContinue(ctx, e.MaxIterations)
				switch result {
				case ConfirmationApprove:
					rt.Resume(ctx, runtime.ResumeTypeApprove)
				case ConfirmationReject:
					rt.Resume(ctx, runtime.ResumeTypeReject)
					return nil
				case ConfirmationAbort:
					rt.Resume(ctx, runtime.ResumeTypeReject)
					return nil
				}
			case *runtime.ElicitationRequestEvent:
				if llmIsTyping {
					fmt.Println()
					llmIsTyping = false
				}

				serverURL := e.Meta["cagent/server_url"].(string)
				result := PromptOAuthAuthorization(ctx, serverURL)
				switch {
				case ctx.Err() != nil:
					return ctx.Err()
				case result == ConfirmationApprove:
					_ = rt.ResumeElicitation(ctx, "accept", nil)
				case result == ConfirmationReject:
					_ = rt.ResumeElicitation(ctx, "decline", nil)
					return fmt.Errorf("OAuth authorization rejected by user")
				}
			}
		}

		// If the loop ended due to Ctrl+C, inform the user succinctly
		if ctx.Err() != nil {
			fmt.Println(Yellow("\n⚠️  agent stopped  ⚠️"))
		}

		// Wrap runtime errors to prevent duplicate error messages and usage display
		if lastErr != nil {
			return RuntimeError{Err: lastErr}
		}
		return nil
	}

	if len(args) == 2 {
		if args[1] == "-" {
			buf, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read from stdin: %w", err)
			}

			if err := oneLoop(string(buf), os.Stdin); err != nil {
				return err
			}
		} else {
			if err := oneLoop(args[1], os.Stdin); err != nil {
				return err
			}
		}
	} else {
		PrintWelcomeMessage(cfg.AppName)
		firstQuestion := true
		for {
			if !firstQuestion {
				fmt.Print("\n\n")
			}
			fmt.Print(Blue("> "))
			firstQuestion = false

			line, err := input.ReadLine(ctx, os.Stdin)
			if err != nil {
				return err
			}

			if err := oneLoop(line, os.Stdin); err != nil {
				return err
			}
		}
	}

	// Wrap runtime errors to prevent duplicate error messages and usage display
	if lastErr != nil {
		return RuntimeError{Err: lastErr}
	}
	return nil
}

// runUserCommand handles built-in session commands
// TODO: This is a duplication of builtInSessionCommands() in pkg/tui/tui.go
func runUserCommand(userInput string, sess *session.Session, rt runtime.Runtime, ctx context.Context) (bool, error) {
	switch userInput {
	case "/exit":
		os.Exit(0)
	case "/eval":
		evalFile, err := evaluation.Save(sess)
		if err == nil {
			fmt.Printf("%s\n", Yellow("Evaluation saved to file %s", evalFile))
			return true, err
		}
		return true, nil
	case "/usage":
		fmt.Printf("%s\n", Yellow("Input tokens: %d", sess.InputTokens))
		fmt.Printf("%s\n", Yellow("Output tokens: %d", sess.OutputTokens))
		return true, nil
	case "/new":
		// Reset session items
		sess.Messages = []session.Item{}
		return true, nil
	case "/compact":
		// Generate a summary of the session and compact the history
		fmt.Printf("%s\n", Yellow("Generating summary..."))

		// Create a channel to capture summary events
		events := make(chan runtime.Event, 100)

		// Generate the summary
		rt.Summarize(ctx, sess, events)

		// Process events and show the summary
		close(events)
		summaryGenerated := false
		hasWarning := false
		for event := range events {
			switch e := event.(type) {
			case *runtime.SessionSummaryEvent:
				fmt.Printf("%s\n", Yellow("Summary generated and added to session"))
				fmt.Printf("Summary: %s\n", e.Summary)
				summaryGenerated = true
			case *runtime.WarningEvent:
				fmt.Printf("%s\n", Yellow("Warning: "+e.Message))
				hasWarning = true
			}
		}

		if !summaryGenerated && !hasWarning {
			fmt.Printf("%s\n", Yellow("No summary generated"))
		}

		return true, nil
	}

	return false, nil
}

// parseAttachCommand parses user input for /attach commands
// Returns the message text (with /attach commands removed) and the attachment path
func parseAttachCommand(userInput string) (messageText, attachPath string) {
	lines := strings.Split(userInput, "\n")
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

	return session.UserMessage(agentFilename, "", multiContent...)
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

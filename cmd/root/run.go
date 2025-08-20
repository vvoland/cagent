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

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/docker/cagent/pkg/chat"
	"github.com/docker/cagent/pkg/content"
	"github.com/docker/cagent/pkg/evaluation"
	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/teamloader"
)

var autoApprove bool
var attachmentPath string

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
	addGatewayFlags(cmd)

	return cmd
}

func runAgentCommand(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	agentFilename := args[0]

	logger := newLogger()

	logger.Debug("Starting agent", "agent", agentName, "debug_mode", debugMode)

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

	rt := runtime.New(logger, agents,
		runtime.WithCurrentAgent(agentName),
		runtime.WithAutoRunTools(autoApprove),
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

		handled, err := runUserCommand(userInput, sess)
		if err != nil {
			return err
		}

		if handled {
			return nil
		}

		sess.Messages = append(sess.Messages, createUserMessageWithAttachment(agentFilename, userInput, attachmentPath))

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

func runUserCommand(userInput string, sess *session.Session) (bool, error) {
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
	case "/reset":
		sess.Messages = []session.Message{}
		return true, nil
	}

	return false, nil
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
func createUserMessageWithAttachment(agentFilename, userContent, attachmentPath string) session.Message {
	if attachmentPath == "" {
		return session.UserMessage(agentFilename, userContent)
	}

	// Convert file to data URL
	dataURL, err := fileToDataURL(attachmentPath)
	if err != nil {
		fmt.Printf("Warning: Failed to attach file %s: %v\n", attachmentPath, err)
		return session.UserMessage(agentFilename, userContent)
	}

	// Create message with multi-content including text and image
	multiContent := []chat.MessagePart{
		{
			Type: chat.MessagePartTypeText,
			Text: userContent,
		},
		{
			Type: chat.MessagePartTypeImageURL,
			ImageURL: &chat.MessageImageURL{
				URL:    dataURL,
				Detail: chat.ImageURLDetailAuto,
			},
		},
	}

	return session.Message{
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

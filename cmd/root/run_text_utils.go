package root

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docker/cagent/pkg/tools"
	"github.com/fatih/color"
)

// text colors
var blue = color.New(color.FgBlue).SprintfFunc()
var yellow = color.New(color.FgYellow).SprintfFunc()
var red = color.New(color.FgRed).SprintfFunc()
var gray = color.New(color.FgHiBlack).SprintfFunc()

// text styles
var bold = color.New(color.Bold).SprintfFunc()

// confirmation result types
type ConfirmationResult string

const (
	ConfirmationApprove        ConfirmationResult = "approve"
	ConfirmationApproveSession ConfirmationResult = "approve_session"
	ConfirmationReject         ConfirmationResult = "reject"
)

// text utility functions

func printWelcomeMessage() {
	fmt.Println(blue("\nWelcome to %s! (Ctrl+C to exit)\n", bold(APP_NAME)))
}

func printError(err error) {
	fmt.Println(red("❌ %s", err))
}

func printAgentName(agentName string) {
	fmt.Printf("\n%s\n", blue("--- Agent: %s ---", bold(agentName)))
}

func printToolCall(toolCall tools.ToolCall, colorFunc ...func(format string, a ...interface{}) string) {
	c := gray
	if len(colorFunc) > 0 && colorFunc[0] != nil {
		c = colorFunc[0]
	}
	fmt.Printf("\n%s\n", c("%s%s", bold(toolCall.Function.Name), formatToolCallArguments(toolCall.Function.Arguments)))
}

func printToolCallWithConfirmation(toolCall tools.ToolCall, scanner *bufio.Scanner) ConfirmationResult {
	fmt.Printf("\n%s\n", bold(yellow("🛠️ Tool call requires confirmation 🛠️")))
	printToolCall(toolCall, color.New(color.FgWhite).SprintfFunc())
	fmt.Printf("\n%s", bold(yellow("Can I run this tool? ([y]es/[a]ll/[n]o): ")))

	scanner.Scan()
	text := scanner.Text()
	switch text {
	case "y":
		return ConfirmationApprove
	case "a":
		return ConfirmationApproveSession
	case "n":
		return ConfirmationReject
	default:
		// Default to reject for invalid input
		return ConfirmationReject
	}
}

func printToolCallResponse(toolCall tools.ToolCall, response string) {
	fmt.Printf("\n%s\n", gray("%s response%s", bold(toolCall.Function.Name), formatToolCallResponse(response)))
}

func formatToolCallArguments(arguments string) string {
	if arguments == "" {
		return "()"
	}

	// Parse JSON to validate it and reformat
	var parsed interface{}
	if err := json.Unmarshal([]byte(arguments), &parsed); err != nil {
		// If JSON parsing fails, return the original string
		return fmt.Sprintf("(%s)", arguments)
	}

	// Custom format that handles multiline strings better
	return formatParsedJSON(parsed)
}

func formatToolCallResponse(response string) string {
	if response == "" {
		return " → ()"
	}

	// For responses, we want to show them as readable text, not JSON
	// Check if it looks like JSON first
	var parsed interface{}
	if err := json.Unmarshal([]byte(response), &parsed); err == nil {
		// It's valid JSON, format it nicely
		return " → " + formatParsedJSON(parsed)
	}

	// It's plain text, handle multiline content
	if strings.Contains(response, "\n") {
		// Trim whitespace and split into lines
		trimmed := strings.TrimSpace(response)
		lines := strings.Split(trimmed, "\n")

		if len(lines) <= 3 {
			// Short multiline, show inline
			return fmt.Sprintf(" → %q", response)
		}

		// Long multiline, format with line breaks
		// Process each line individually and collapse consecutive empty lines
		var formatted []string
		lastWasEmpty := false

		for _, line := range lines {
			trimmedLine := strings.TrimSpace(line)
			if trimmedLine == "" {
				// Empty line - only add one if the last line wasn't empty
				if !lastWasEmpty {
					formatted = append(formatted, "")
					lastWasEmpty = true
				}
			} else {
				formatted = append(formatted, line)
				lastWasEmpty = false
			}
		}
		return fmt.Sprintf(" → (\n%s\n)", strings.Join(formatted, "\n"))
	}

	// Single line text response
	return fmt.Sprintf(" → %q", response)
}

func formatParsedJSON(data interface{}) string {
	switch v := data.(type) {
	case map[string]interface{}:
		if len(v) == 0 {
			return "()"
		}

		parts := make([]string, 0, len(v))
		hasMultilineContent := false

		for key, value := range v {
			formatted := formatJSONValue(key, value)
			parts = append(parts, formatted)
			if strings.Contains(formatted, "\n") {
				hasMultilineContent = true
			}
		}

		if len(parts) == 1 && !hasMultilineContent {
			return fmt.Sprintf("(%s)", parts[0])
		}

		return fmt.Sprintf("(\n  %s\n)", strings.Join(parts, "\n  "))

	default:
		// For non-object types, use standard JSON formatting
		formatted, _ := json.MarshalIndent(data, "", "  ")
		return fmt.Sprintf("(%s)", string(formatted))
	}
}

func formatJSONValue(key string, value interface{}) string {
	switch v := value.(type) {
	case string:
		// Handle multiline strings by displaying with actual newlines
		if strings.Contains(v, "\n") {
			// Format as: key: "content with
			//              actual line breaks"
			return fmt.Sprintf("%s: \"%s\"", bold(key), v)
		}
		// Regular string with proper escaping
		return fmt.Sprintf("%s: %q", bold(key), v)

	case []interface{}:
		if len(v) == 0 {
			return fmt.Sprintf("%s: []", bold(key))
		}
		// Show full array contents
		jsonBytes, _ := json.MarshalIndent(v, "", "  ")
		return fmt.Sprintf("%s: %s", bold(key), string(jsonBytes))

	case map[string]interface{}:
		jsonBytes, _ := json.MarshalIndent(v, "", "  ")
		return fmt.Sprintf("%s: %s", bold(key), string(jsonBytes))

	default:
		jsonBytes, _ := json.Marshal(v)
		return fmt.Sprintf("%s: %s", bold(key), string(jsonBytes))
	}
}

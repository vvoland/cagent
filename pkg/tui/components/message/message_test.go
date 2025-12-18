package message

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/pkg/tui/types"
)

func TestErrorMessageWrapping(t *testing.T) {
	t.Parallel()

	// Create a long error message that should wrap
	longError := "This is a very long error message that should wrap to multiple lines when the width is constrained. " +
		"It contains enough text to exceed typical terminal widths and demonstrate the wrapping behavior."

	msg := types.Error(longError)
	mv := New(msg, nil)

	// Set a narrow width to force wrapping
	width := 50
	mv.SetSize(width, 0)

	// Render the message
	rendered := mv.View()

	// Verify that the message was rendered
	require.NotEmpty(t, rendered)

	// Verify that the content was wrapped (should have multiple lines)
	lines := strings.Split(rendered, "\n")
	assert.Greater(t, len(lines), 1, "Error message should wrap to multiple lines")

	// Verify each line respects the width constraint (accounting for borders and padding)
	for i, line := range lines {
		// Strip ANSI codes for accurate width calculation
		plainLine := stripANSI(line)
		// Allow some flexibility for borders and padding
		assert.LessOrEqual(t, len(plainLine), width+10, "Line %d exceeds width constraint: %q", i, plainLine)
	}
}

func TestErrorMessageWithShortContent(t *testing.T) {
	t.Parallel()

	shortError := "Short error"
	msg := types.Error(shortError)
	mv := New(msg, nil)

	width := 80
	mv.SetSize(width, 0)

	rendered := mv.View()

	// Verify that the message was rendered
	require.NotEmpty(t, rendered)

	// Verify the content is present in the rendered output
	plainRendered := stripANSI(rendered)
	assert.Contains(t, plainRendered, shortError)
}

func TestErrorMessagePreservesContent(t *testing.T) {
	t.Parallel()

	errorContent := "Error: Failed to connect to database\nConnection timeout after 30 seconds"
	msg := types.Error(errorContent)
	mv := New(msg, nil)

	width := 80
	mv.SetSize(width, 0)

	rendered := mv.View()

	// Verify that the message was rendered
	require.NotEmpty(t, rendered)

	// Verify the essential content is preserved (may be reformatted but words should be there)
	plainRendered := stripANSI(rendered)
	assert.Contains(t, plainRendered, "Failed to connect")
	assert.Contains(t, plainRendered, "database")
	assert.Contains(t, plainRendered, "timeout")
}

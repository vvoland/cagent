package styles

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// ANSI reset sequences we need to handle
const (
	resetFull  = "\x1b[0m"
	resetShort = "\x1b[m"
)

// RenderComposite renders the content with the given style, but ensures that
// any ANSI reset codes in the content are replaced with the style's active sequences,
// preventing the style's background/foreground from being interrupted.
func RenderComposite(style lipgloss.Style, content string) string {
	// Fast path: if content has no reset sequences, just render normally
	if !strings.Contains(content, "\x1b[") {
		return style.Render(content)
	}

	// Get the escape sequence for the style without layout (padding/margin)
	// We render an empty string to get the codes.
	// output is usually: <codes><reset>
	cleanStyle := style.
		UnsetPadding().
		UnsetMargins().
		UnsetWidth().
		UnsetHeight().
		UnsetBold().
		UnsetItalic().
		UnsetUnderline().
		UnsetStrikethrough().
		UnsetReverse().
		UnsetBlink().
		UnsetFaint().
		UnsetBorderStyle().
		UnsetBorderTop().
		UnsetBorderBottom().
		UnsetBorderLeft().
		UnsetBorderRight().
		UnsetBorderForeground().
		UnsetBorderBackground()
	styleSeq := cleanStyle.Render("")

	// Remove the trailing reset code to get just the "start" sequence
	// We assume the last sequence is the reset.
	// lipgloss typically ends with \x1b[0m or \x1b[m
	styleSeq = strings.TrimSuffix(styleSeq, resetFull)
	styleSeq = strings.TrimSuffix(styleSeq, resetShort)

	// Replace reset sequences with reset + styleSeq
	// Handle both \x1b[0m and \x1b[m forms without regex
	modifiedContent := strings.ReplaceAll(content, resetFull, resetFull+styleSeq)
	// Only replace short form if it wasn't already replaced as part of full form
	// The short form \x1b[m is a subset pattern, so we need to be careful
	// Actually, they are distinct: \x1b[0m vs \x1b[m, so we can replace both
	modifiedContent = strings.ReplaceAll(modifiedContent, resetShort, resetFull+styleSeq)

	// Render the modified content with the original style (to keep padding/layout)
	return style.Render(modifiedContent)
}

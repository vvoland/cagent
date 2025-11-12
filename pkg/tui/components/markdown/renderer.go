package markdown

import (
	"github.com/charmbracelet/glamour/v2"

	"github.com/docker/cagent/pkg/tui/styles"
)

func NewRenderer(width int) *glamour.TermRenderer {
	style := styles.MarkdownStyle()

	r, _ := glamour.NewTermRenderer(
		glamour.WithWordWrap(width),
		glamour.WithStyles(style),
	)
	return r
}

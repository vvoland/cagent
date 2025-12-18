package tab

import (
	"strings"

	"github.com/docker/cagent/pkg/tui/styles"
)

func Render(title, content string, width int) string {
	var b strings.Builder

	b.WriteString(styles.RenderComposite(styles.TabTitleStyle, title+" "+strings.Repeat("─", width-len(title)-1)))
	b.WriteString("\n")
	b.WriteString(styles.RenderComposite(styles.TabStyle.Width(width-2), content))
	b.WriteString("\n")

	return b.String()
}

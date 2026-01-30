package dialog

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/tui/core"
	"github.com/docker/cagent/pkg/tui/core/layout"
	"github.com/docker/cagent/pkg/tui/styles"
)

// permissionsDialog displays the configured tool permissions (allow/deny patterns).
type permissionsDialog struct {
	BaseDialog
	permissions *runtime.PermissionsInfo
	keyMap      permissionsDialogKeyMap
	offset      int
}

type permissionsDialogKeyMap struct {
	Close, Up, Down, PageUp, PageDown key.Binding
}

// NewPermissionsDialog creates a new dialog showing tool permission rules.
func NewPermissionsDialog(perms *runtime.PermissionsInfo) Dialog {
	return &permissionsDialog{
		permissions: perms,
		keyMap: permissionsDialogKeyMap{
			Close:    key.NewBinding(key.WithKeys("esc", "enter", "q"), key.WithHelp("Esc", "close")),
			Up:       key.NewBinding(key.WithKeys("up", "k")),
			Down:     key.NewBinding(key.WithKeys("down", "j")),
			PageUp:   key.NewBinding(key.WithKeys("pgup")),
			PageDown: key.NewBinding(key.WithKeys("pgdown")),
		},
	}
}

func (d *permissionsDialog) Init() tea.Cmd {
	return nil
}

func (d *permissionsDialog) Update(msg tea.Msg) (layout.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cmd := d.SetSize(msg.Width, msg.Height)
		return d, cmd

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, d.keyMap.Close):
			return d, core.CmdHandler(CloseDialogMsg{})
		case key.Matches(msg, d.keyMap.Up):
			d.offset = max(0, d.offset-1)
		case key.Matches(msg, d.keyMap.Down):
			d.offset++
		case key.Matches(msg, d.keyMap.PageUp):
			d.offset = max(0, d.offset-d.pageSize())
		case key.Matches(msg, d.keyMap.PageDown):
			d.offset += d.pageSize()
		}

	case tea.MouseWheelMsg:
		switch msg.Button.String() {
		case "wheelup":
			d.offset = max(0, d.offset-1)
		case "wheeldown":
			d.offset++
		}
	}
	return d, nil
}

func (d *permissionsDialog) dialogSize() (dialogWidth, maxHeight, contentWidth int) {
	dialogWidth = d.ComputeDialogWidth(60, 40, 70)
	maxHeight = min(d.Height()*70/100, 30)
	contentWidth = d.ContentWidth(dialogWidth, 2)
	return dialogWidth, maxHeight, contentWidth
}

func (d *permissionsDialog) pageSize() int {
	_, maxHeight, _ := d.dialogSize()
	return max(1, maxHeight-10)
}

func (d *permissionsDialog) Position() (row, col int) {
	dialogWidth, maxHeight, _ := d.dialogSize()
	return CenterPosition(d.Width(), d.Height(), dialogWidth, maxHeight)
}

func (d *permissionsDialog) View() string {
	dialogWidth, maxHeight, contentWidth := d.dialogSize()
	content := d.renderContent(contentWidth, maxHeight)
	return styles.DialogStyle.Padding(1, 2).Width(dialogWidth).Render(content)
}

func (d *permissionsDialog) renderContent(contentWidth, maxHeight int) string {
	// Build all lines
	lines := []string{
		RenderTitle("Tool Permissions", contentWidth, styles.DialogTitleStyle),
		RenderSeparator(contentWidth),
		"",
	}

	if d.permissions == nil {
		lines = append(lines, styles.MutedStyle.Render("No permissions configured."), "")
	} else {
		// Deny section (checked first during evaluation)
		if len(d.permissions.Deny) > 0 {
			lines = append(lines, d.renderSectionHeader("Deny", "These tools are always blocked"), "")
			for _, pattern := range d.permissions.Deny {
				lines = append(lines, d.renderPattern(pattern, true))
			}
			lines = append(lines, "")
		}

		// Allow section
		if len(d.permissions.Allow) > 0 {
			lines = append(lines, d.renderSectionHeader("Allow", "These tools are auto-approved"), "")
			for _, pattern := range d.permissions.Allow {
				lines = append(lines, d.renderPattern(pattern, false))
			}
			lines = append(lines, "")
		}

		// If both are empty
		if len(d.permissions.Allow) == 0 && len(d.permissions.Deny) == 0 {
			lines = append(lines, styles.MutedStyle.Render("No permission patterns configured."), "")
		}
	}

	// Add help about default behavior
	lines = append(lines, styles.MutedStyle.Render("Tools not matching any pattern require user approval."), "")

	// Apply scrolling
	return d.applyScrolling(lines, contentWidth, maxHeight)
}

func (d *permissionsDialog) renderSectionHeader(title, description string) string {
	header := lipgloss.NewStyle().Bold(true).Foreground(styles.TextSecondary).Render(title)
	desc := styles.MutedStyle.Render(" - " + description)
	return header + desc
}

func (d *permissionsDialog) renderPattern(pattern string, isDeny bool) string {
	// Use different colors for deny (red-ish) vs allow (green-ish)
	var icon string
	var style lipgloss.Style
	if isDeny {
		icon = "✗"
		style = lipgloss.NewStyle().Foreground(styles.Error)
	} else {
		icon = "✓"
		style = lipgloss.NewStyle().Foreground(styles.Success)
	}

	return style.Render(icon) + "  " + lipgloss.NewStyle().Foreground(styles.Highlight).Render(pattern)
}

func (d *permissionsDialog) applyScrolling(allLines []string, contentWidth, maxHeight int) string {
	const headerLines = 3 // title + separator + space
	const footerLines = 2 // space + help

	visibleLines := max(1, maxHeight-headerLines-footerLines-4)
	contentLines := allLines[headerLines:]
	totalContentLines := len(contentLines)

	// Clamp offset
	maxOffset := max(0, totalContentLines-visibleLines)
	d.offset = min(d.offset, maxOffset)

	// Extract visible portion
	endIdx := min(d.offset+visibleLines, totalContentLines)
	parts := append(allLines[:headerLines], contentLines[d.offset:endIdx]...)

	// Scroll indicator
	if totalContentLines > visibleLines {
		scrollInfo := lipgloss.NewStyle().Render("")
		if d.offset > 0 {
			scrollInfo = "↑ "
		}
		scrollInfo += styles.MutedStyle.Render("[" + strings.Repeat("─", 3) + "]")
		if endIdx < totalContentLines {
			scrollInfo += " ↓"
		}
		parts = append(parts, styles.MutedStyle.Render(scrollInfo))
	}

	parts = append(parts, "", RenderHelpKeys(contentWidth, "↑↓", "scroll", "Esc", "close"))
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

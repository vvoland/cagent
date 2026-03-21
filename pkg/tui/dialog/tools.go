package dialog

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/docker/docker-agent/pkg/tools"
	"github.com/docker/docker-agent/pkg/tui/components/scrollview"
	"github.com/docker/docker-agent/pkg/tui/components/toolcommon"
	"github.com/docker/docker-agent/pkg/tui/core"
	"github.com/docker/docker-agent/pkg/tui/core/layout"
	"github.com/docker/docker-agent/pkg/tui/styles"
)

// toolsDialog displays all tools available to the current agent.
type toolsDialog struct {
	BaseDialog

	tools      []tools.Tool
	closeKey   key.Binding
	scrollview *scrollview.Model
}

// NewToolsDialog creates a new dialog showing all available tools.
func NewToolsDialog(toolList []tools.Tool) Dialog {
	// Sort tools by category then name
	sorted := make([]tools.Tool, len(toolList))
	copy(sorted, toolList)
	slices.SortFunc(sorted, func(a, b tools.Tool) int {
		if c := strings.Compare(strings.ToLower(a.Category), strings.ToLower(b.Category)); c != 0 {
			return c
		}
		return strings.Compare(strings.ToLower(a.DisplayName()), strings.ToLower(b.DisplayName()))
	})

	return &toolsDialog{
		tools: sorted,
		scrollview: scrollview.New(
			scrollview.WithKeyMap(scrollview.ReadOnlyScrollKeyMap()),
			scrollview.WithReserveScrollbarSpace(true),
		),
		closeKey: key.NewBinding(key.WithKeys("esc", "enter", "q"), key.WithHelp("Esc", "close")),
	}
}

func (d *toolsDialog) Init() tea.Cmd {
	return nil
}

func (d *toolsDialog) Update(msg tea.Msg) (layout.Model, tea.Cmd) {
	if handled, cmd := d.scrollview.Update(msg); handled {
		return d, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cmd := d.SetSize(msg.Width, msg.Height)
		return d, cmd

	case tea.KeyPressMsg:
		if key.Matches(msg, d.closeKey) {
			return d, core.CmdHandler(CloseDialogMsg{})
		}
	}
	return d, nil
}

func (d *toolsDialog) dialogSize() (dialogWidth, maxHeight, contentWidth int) {
	dialogWidth = d.ComputeDialogWidth(70, 50, 80)
	maxHeight = min(d.Height()*80/100, 40)
	contentWidth = d.ContentWidth(dialogWidth, 2) - d.scrollview.ReservedCols()
	return dialogWidth, maxHeight, contentWidth
}

func (d *toolsDialog) Position() (row, col int) {
	dialogWidth, maxHeight, _ := d.dialogSize()
	return CenterPosition(d.Width(), d.Height(), dialogWidth, maxHeight)
}

func (d *toolsDialog) View() string {
	dialogWidth, maxHeight, contentWidth := d.dialogSize()
	content := d.renderContent(contentWidth, maxHeight)
	return styles.DialogStyle.Padding(1, 2).Width(dialogWidth).Render(content)
}

func (d *toolsDialog) renderContent(contentWidth, maxHeight int) string {
	title := fmt.Sprintf("Tools (%d)", len(d.tools))
	lines := []string{
		RenderTitle(title, contentWidth, styles.DialogTitleStyle),
		RenderSeparator(contentWidth),
		"",
	}

	if len(d.tools) == 0 {
		lines = append(lines, styles.MutedStyle.Render("No tools available."), "")
	} else {
		var lastCategory string
		for i := range d.tools {
			t := &d.tools[i]
			cat := t.Category
			if cat == "" {
				cat = "Other"
			}
			if cat != lastCategory {
				if lastCategory != "" {
					lines = append(lines, "")
				}
				lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(styles.TextSecondary).Render(cat))
				lastCategory = cat
			}

			name := lipgloss.NewStyle().Foreground(styles.Highlight).Render("  " + t.DisplayName())
			if t.Description != "" {
				separator := " • "
				separatorWidth := lipgloss.Width(separator)
				nameWidth := lipgloss.Width(name)
				availableWidth := contentWidth - nameWidth - separatorWidth
				if availableWidth > 0 {
					desc := toolcommon.TruncateText(t.Description, availableWidth)
					name += styles.MutedStyle.Render(separator + desc)
				}
			}
			lines = append(lines, name)
		}
		lines = append(lines, "")
	}

	return d.applyScrolling(lines, contentWidth, maxHeight)
}

func (d *toolsDialog) applyScrolling(allLines []string, contentWidth, maxHeight int) string {
	const headerLines = 3 // title + separator + space
	const footerLines = 2 // space + help

	visibleLines := max(1, maxHeight-headerLines-footerLines-4)
	contentLines := allLines[headerLines:]

	regionWidth := contentWidth + d.scrollview.ReservedCols()
	d.scrollview.SetSize(regionWidth, visibleLines)

	dialogRow, dialogCol := d.Position()
	d.scrollview.SetPosition(dialogCol+3, dialogRow+2+headerLines)

	d.scrollview.SetContent(contentLines, len(contentLines))

	scrollableContent := d.scrollview.View()
	parts := append(allLines[:headerLines], scrollableContent)
	parts = append(parts, "", RenderHelpKeys(regionWidth, "↑↓", "scroll", "Esc", "close"))
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

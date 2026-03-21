package dialog

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/docker/docker-agent/pkg/tui/components/scrollview"
	"github.com/docker/docker-agent/pkg/tui/core"
	"github.com/docker/docker-agent/pkg/tui/core/layout"
	"github.com/docker/docker-agent/pkg/tui/styles"
)

// readOnlyScrollDialogSize defines the sizing parameters for a read-only scroll dialog.
type readOnlyScrollDialogSize struct {
	widthPercent  int
	minWidth      int
	maxWidth      int
	heightPercent int
	heightMax     int
}

// contentRenderer renders dialog content lines given the available width and max height.
type contentRenderer func(contentWidth, maxHeight int) []string

// readOnlyScrollDialog is a base for simple read-only dialogs with scrollable content.
// It handles Init, Update (scrollview + close key), Position, View, and scrolling.
// Concrete dialogs embed it and provide a contentRenderer and help key bindings.
type readOnlyScrollDialog struct {
	BaseDialog

	scrollview *scrollview.Model
	closeKey   key.Binding
	size       readOnlyScrollDialogSize
	render     contentRenderer
	helpKeys   []string // pairs of [key, description] for the footer
}

// newReadOnlyScrollDialog creates a new read-only scrollable dialog.
func newReadOnlyScrollDialog(
	size readOnlyScrollDialogSize,
	render contentRenderer,
) readOnlyScrollDialog {
	return readOnlyScrollDialog{
		scrollview: scrollview.New(
			scrollview.WithKeyMap(scrollview.ReadOnlyScrollKeyMap()),
			scrollview.WithReserveScrollbarSpace(true),
		),
		closeKey: key.NewBinding(key.WithKeys("esc", "enter", "q"), key.WithHelp("Esc", "close")),
		size:     size,
		render:   render,
		helpKeys: []string{"↑↓", "scroll", "Esc", "close"},
	}
}

func (d *readOnlyScrollDialog) Init() tea.Cmd {
	return nil
}

func (d *readOnlyScrollDialog) Update(msg tea.Msg) (layout.Model, tea.Cmd) {
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

func (d *readOnlyScrollDialog) dialogSize() (dialogWidth, maxHeight, contentWidth int) {
	s := d.size
	dialogWidth = d.ComputeDialogWidth(s.widthPercent, s.minWidth, s.maxWidth)
	maxHeight = min(d.Height()*s.heightPercent/100, s.heightMax)
	contentWidth = d.ContentWidth(dialogWidth, 2) - d.scrollview.ReservedCols()
	return dialogWidth, maxHeight, contentWidth
}

func (d *readOnlyScrollDialog) Position() (row, col int) {
	dialogWidth, maxHeight, _ := d.dialogSize()
	return CenterPosition(d.Width(), d.Height(), dialogWidth, maxHeight)
}

func (d *readOnlyScrollDialog) View() string {
	dialogWidth, maxHeight, contentWidth := d.dialogSize()
	allLines := d.render(contentWidth, maxHeight)

	const headerLines = 3 // title + separator + space
	contentLines := allLines[headerLines:]

	regionWidth := contentWidth + d.scrollview.ReservedCols()
	visibleLines := max(1, maxHeight-headerLines-2-4) // 2 = footer (space + help), 4 = dialog chrome
	d.scrollview.SetSize(regionWidth, visibleLines)

	dialogRow, dialogCol := d.Position()
	d.scrollview.SetPosition(dialogCol+3, dialogRow+2+headerLines)
	d.scrollview.SetContent(contentLines, len(contentLines))

	parts := append(allLines[:headerLines], d.scrollview.View())
	parts = append(parts, "", RenderHelpKeys(regionWidth, d.helpKeys...))

	content := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return styles.DialogStyle.Padding(1, 2).Width(dialogWidth).Render(content)
}

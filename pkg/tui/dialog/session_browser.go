package dialog

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/tui/core"
	"github.com/docker/cagent/pkg/tui/core/layout"
	"github.com/docker/cagent/pkg/tui/messages"
	"github.com/docker/cagent/pkg/tui/styles"
)

type sessionBrowserDialog struct {
	BaseDialog
	textInput textinput.Model
	sessions  []session.Summary
	filtered  []session.Summary
	selected  int
	offset    int                  // scroll offset for viewport
	keyMap    commandPaletteKeyMap // Reuse existing keymap
	openedAt  time.Time            // when dialog was opened, for stable time display
}

// NewSessionBrowserDialog creates a new session browser dialog
func NewSessionBrowserDialog(sessions []session.Summary) Dialog {
	ti := textinput.New()
	ti.Placeholder = "Type to search sessions…"
	ti.Focus()
	ti.CharLimit = 100
	ti.SetWidth(50)

	return &sessionBrowserDialog{
		textInput: ti,
		sessions:  sessions,
		filtered:  sessions,
		keyMap:    defaultCommandPaletteKeyMap(),
		openedAt:  time.Now(),
	}
}

func (d *sessionBrowserDialog) Init() tea.Cmd {
	return textinput.Blink
}

func (d *sessionBrowserDialog) Update(msg tea.Msg) (layout.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cmd := d.SetSize(msg.Width, msg.Height)
		return d, cmd

	case tea.KeyPressMsg:
		if cmd := HandleQuit(msg); cmd != nil {
			return d, cmd
		}

		switch {
		case key.Matches(msg, d.keyMap.Escape):
			return d, core.CmdHandler(CloseDialogMsg{})

		case key.Matches(msg, d.keyMap.Up):
			if d.selected > 0 {
				d.selected--
			}
			return d, nil

		case key.Matches(msg, d.keyMap.Down):
			if d.selected < len(d.filtered)-1 {
				d.selected++
			}
			return d, nil

		case key.Matches(msg, d.keyMap.PageUp):
			d.selected -= d.pageSize()
			if d.selected < 0 {
				d.selected = 0
			}
			return d, nil

		case key.Matches(msg, d.keyMap.PageDown):
			d.selected += d.pageSize()
			if d.selected >= len(d.filtered) {
				d.selected = max(0, len(d.filtered)-1)
			}
			return d, nil

		case key.Matches(msg, d.keyMap.Enter):
			if d.selected >= 0 && d.selected < len(d.filtered) {
				return d, tea.Sequence(
					core.CmdHandler(CloseDialogMsg{}),
					core.CmdHandler(messages.LoadSessionMsg{SessionID: d.filtered[d.selected].ID}),
				)
			}
			return d, nil

		default:
			var cmd tea.Cmd
			d.textInput, cmd = d.textInput.Update(msg)
			d.filterSessions()
			return d, cmd
		}
	}

	return d, nil
}

func (d *sessionBrowserDialog) filterSessions() {
	query := strings.ToLower(strings.TrimSpace(d.textInput.Value()))
	if query == "" {
		d.filtered = d.sessions
		d.selected = 0
		d.offset = 0
		return
	}

	d.filtered = nil
	for _, sess := range d.sessions {
		title := sess.Title
		if title == "" {
			title = "Untitled"
		}
		if strings.Contains(strings.ToLower(title), query) {
			d.filtered = append(d.filtered, sess)
		}
	}

	if d.selected >= len(d.filtered) {
		d.selected = 0
	}
	d.offset = 0
}

func (d *sessionBrowserDialog) dialogSize() (dialogWidth, maxHeight, contentWidth int) {
	dialogWidth = max(min(d.Width()*80/100, 80), 60)
	maxHeight = min(d.Height()*70/100, 30)
	contentWidth = dialogWidth - 6
	return dialogWidth, maxHeight, contentWidth
}

func (d *sessionBrowserDialog) View() string {
	dialogWidth, maxHeight, contentWidth := d.dialogSize()

	d.textInput.SetWidth(contentWidth)

	var sessionLines []string
	maxItems := maxHeight - 8

	// Adjust offset to keep selected item visible
	if d.selected < d.offset {
		d.offset = d.selected
	} else if d.selected >= d.offset+maxItems {
		d.offset = d.selected - maxItems + 1
	}

	// Render visible items based on offset
	visibleEnd := min(d.offset+maxItems, len(d.filtered))
	for i := d.offset; i < visibleEnd; i++ {
		sessionLines = append(sessionLines, d.renderSession(d.filtered[i], i == d.selected, contentWidth))
	}

	// Show indicator if there are more items
	if visibleEnd < len(d.filtered) {
		sessionLines = append(sessionLines, styles.MutedStyle.Render(fmt.Sprintf("  … and %d more", len(d.filtered)-visibleEnd)))
	}

	if len(d.filtered) == 0 {
		sessionLines = append(sessionLines, "", styles.DialogContentStyle.
			Italic(true).
			Align(lipgloss.Center).
			Width(contentWidth).
			Render("No sessions found"))
	}

	content := NewContent(contentWidth).
		AddTitle("Sessions").
		AddSpace().
		AddContent(d.textInput.View()).
		AddSeparator().
		AddContent(strings.Join(sessionLines, "\n")).
		AddSpace().
		AddHelp("↑/↓ navigate • pgup/pgdn page • enter load • esc close").
		Build()

	return styles.DialogStyle.Width(dialogWidth).Render(content)
}

func (d *sessionBrowserDialog) pageSize() int {
	_, maxHeight, _ := d.dialogSize()
	return max(1, maxHeight-8)
}

func (d *sessionBrowserDialog) renderSession(sess session.Summary, selected bool, maxWidth int) string {
	titleStyle, timeStyle := styles.PaletteUnselectedActionStyle, styles.PaletteUnselectedDescStyle
	if selected {
		titleStyle, timeStyle = styles.PaletteSelectedActionStyle, styles.PaletteSelectedDescStyle
	}

	title := sess.Title
	if title == "" {
		title = "Untitled"
	}

	maxTitleLen := maxWidth - 25
	if len(title) > maxTitleLen {
		title = title[:maxTitleLen-1] + "…"
	}

	return titleStyle.Render(" "+title) + timeStyle.Render(" • "+d.timeAgo(sess.CreatedAt))
}

func (d *sessionBrowserDialog) timeAgo(t time.Time) string {
	elapsed := d.openedAt.Sub(t)
	switch {
	case elapsed < time.Minute:
		return fmt.Sprintf("%ds ago", int(elapsed.Seconds()))
	case elapsed < time.Hour:
		return fmt.Sprintf("%dm ago", int(elapsed.Minutes()))
	case elapsed < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(elapsed.Hours()))
	case elapsed < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(elapsed.Hours()/24))
	default:
		return t.Format("Jan 2")
	}
}

func (d *sessionBrowserDialog) Position() (row, col int) {
	dialogWidth, maxHeight, _ := d.dialogSize()
	return CenterPosition(d.Width(), d.Height(), dialogWidth, maxHeight)
}

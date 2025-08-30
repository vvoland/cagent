package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"

	"github.com/docker/cagent/internal/app"
	"github.com/docker/cagent/internal/tui/core"
	"github.com/docker/cagent/internal/tui/core/layout"
	"github.com/docker/cagent/internal/tui/dialog"
	"github.com/docker/cagent/internal/tui/page"
	chatpage "github.com/docker/cagent/internal/tui/page/chat"
	"github.com/docker/cagent/internal/tui/styles"
)

var lastMouseEvent time.Time

// MouseEventFilter filters mouse events to prevent spam
func MouseEventFilter(m tea.Model, msg tea.Msg) tea.Msg {
	switch msg.(type) {
	case tea.MouseWheelMsg, tea.MouseMotionMsg:
		now := time.Now()
		if now.Sub(lastMouseEvent) < 20*time.Millisecond {
			return nil
		}
		lastMouseEvent = now
	}
	return msg
}

// appModel represents the main application model
type appModel struct {
	wWidth, wHeight int // Window dimensions
	width, height   int
	keyMap          KeyMap

	currentPage  page.ID
	previousPage page.ID
	pages        map[page.ID]layout.Model
	loadedPages  map[page.ID]bool

	// Dialog system
	dialog dialog.Manager

	// State
	ready bool
	err   error
}

// KeyMap defines global key bindings
type KeyMap struct {
	Quit         key.Binding
	Help         key.Binding
	pageBindings []key.Binding
}

// DefaultKeyMap returns the default global key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("ctrl+g"),
			key.WithHelp("ctrl+g", "help"),
		),
		pageBindings: []key.Binding{},
	}
}

// New creates and initializes a new TUI application model
func New(a *app.App) tea.Model {
	chatPageInstance := chatpage.New(a)
	keyMap := DefaultKeyMap()

	model := &appModel{
		currentPage: chatpage.ChatPageID,
		loadedPages: make(map[page.ID]bool),
		keyMap:      keyMap,
		dialog:      dialog.New(),

		pages: map[page.ID]layout.Model{
			chatpage.ChatPageID: chatPageInstance,
		},
	}

	return model
}

// Init initializes the application
func (a *appModel) Init() tea.Cmd {
	var cmds []tea.Cmd

	// Initialize dialog system
	cmd := a.dialog.Init()
	cmds = append(cmds, cmd)

	// Initialize current page
	cmd = a.pages[a.currentPage].Init()
	cmds = append(cmds, cmd)
	a.loadedPages[a.currentPage] = true

	// Mouse support is configured via program options (cell motion + filter)

	return tea.Batch(cmds...)
}

// Update handles incoming messages and updates the application state
func (a *appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// Handle dialog-specific messages first
	case dialog.OpenDialogMsg, dialog.CloseDialogMsg:
		u, dialogCmd := a.dialog.Update(msg)
		a.dialog = u.(dialog.Manager)
		return a, dialogCmd

	case tea.KeyboardEnhancementsMsg:
		// Forward to all pages and dialogs
		var cmds []tea.Cmd
		for id, page := range a.pages {
			m, pageCmd := page.Update(msg)
			a.pages[id] = m.(layout.Model)
			if pageCmd != nil {
				cmds = append(cmds, pageCmd)
			}
		}
		// Also forward to dialog system
		u, dialogCmd := a.dialog.Update(msg)
		a.dialog = u.(dialog.Manager)
		if dialogCmd != nil {
			cmds = append(cmds, dialogCmd)
		}
		return a, tea.Batch(cmds...)

	case tea.WindowSizeMsg:
		a.wWidth, a.wHeight = msg.Width, msg.Height
		cmd := a.handleWindowResize(msg.Width, msg.Height)
		return a, cmd

	case page.ChangeMsg:
		cmd := a.moveToPage(msg.ID)
		return a, cmd

	case tea.KeyPressMsg:
		cmd := a.handleKeyPressMsg(msg)
		return a, cmd

	case tea.MouseWheelMsg, tea.MouseClickMsg, tea.MouseMotionMsg, tea.MouseReleaseMsg:
		// If dialogs are active, they get priority for mouse events
		if a.dialog.HasDialog() {
			u, dialogCmd := a.dialog.Update(msg)
			a.dialog = u.(dialog.Manager)
			return a, dialogCmd
		}
		// Otherwise forward to current page
		item, ok := a.pages[a.currentPage]
		if !ok {
			return a, nil
		}
		updated, cmd := item.Update(msg)
		a.pages[a.currentPage] = updated.(layout.Model)
		return a, cmd

	case tea.PasteMsg:
		// If dialogs are active, they get priority for paste events
		if a.dialog.HasDialog() {
			u, dialogCmd := a.dialog.Update(msg)
			a.dialog = u.(dialog.Manager)
			return a, dialogCmd
		}
		// Otherwise forward to current page
		item, ok := a.pages[a.currentPage]
		if !ok {
			return a, nil
		}
		updated, cmd := item.Update(msg)
		a.pages[a.currentPage] = updated.(layout.Model)
		return a, cmd

	case error:
		a.err = msg
		return a, nil

	default:
		// For other messages, check if dialogs should handle them first
		// If dialogs are active, they get priority for input
		if a.dialog.HasDialog() {
			u, dialogCmd := a.dialog.Update(msg)
			a.dialog = u.(dialog.Manager)
			return a, dialogCmd
		}

		// Otherwise, forward to current page
		item, ok := a.pages[a.currentPage]
		if !ok {
			return a, nil
		}
		updated, cmd := item.Update(msg)
		a.pages[a.currentPage] = updated.(layout.Model)
		return a, cmd
	}
}

// handleWindowResize processes window resize events
func (a *appModel) handleWindowResize(width, height int) tea.Cmd {
	var cmds []tea.Cmd

	// Update dimensions
	a.width, a.height = width, height-2 // Account for status bar

	if !a.ready {
		a.ready = true
	}

	// Update dialog system
	u, cmd := a.dialog.Update(tea.WindowSizeMsg{Width: width, Height: height})
	a.dialog = u.(dialog.Manager)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	// Update all pages
	for p, page := range a.pages {
		if sizable, ok := page.(interface{ SetSize(int, int) tea.Cmd }); ok {
			cmd := sizable.SetSize(a.width, a.height)
			cmds = append(cmds, cmd)
		} else {
			// Fallback: send window size message
			updated, cmd := page.Update(tea.WindowSizeMsg{Width: a.width, Height: a.height})
			a.pages[p] = updated.(layout.Model)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	return tea.Batch(cmds...)
}

// handleKeyPressMsg processes keyboard input
func (a *appModel) handleKeyPressMsg(msg tea.KeyPressMsg) tea.Cmd {
	// If dialogs are active, they handle key input first
	if a.dialog.HasDialog() {
		u, dialogCmd := a.dialog.Update(msg)
		a.dialog = u.(dialog.Manager)
		return dialogCmd
	}

	switch {
	case key.Matches(msg, a.keyMap.Quit):
		return tea.Quit
	case key.Matches(msg, a.keyMap.Help):
		// Toggle help (would be implemented with a help system)
		return nil
	default:
		// Forward to current page
		item, ok := a.pages[a.currentPage]
		if !ok {
			return nil
		}

		updated, cmd := item.Update(msg)
		a.pages[a.currentPage] = updated.(layout.Model)
		return cmd
	}
}

// moveToPage handles navigation between different pages
func (a *appModel) moveToPage(pageID page.ID) tea.Cmd {
	var cmds []tea.Cmd

	// Initialize page if not loaded
	if _, ok := a.loadedPages[pageID]; !ok {
		cmd := a.pages[pageID].Init()
		cmds = append(cmds, cmd)
		a.loadedPages[pageID] = true
	}

	// Switch pages
	a.previousPage = a.currentPage
	a.currentPage = pageID

	// Set page size
	if sizable, ok := a.pages[a.currentPage].(interface{ SetSize(int, int) tea.Cmd }); ok {
		cmd := sizable.SetSize(a.width, a.height)
		cmds = append(cmds, cmd)
	}

	return tea.Batch(cmds...)
}

// View renders the complete application interface
func (a *appModel) View() string {
	// Handle minimum window size
	if a.wWidth < 25 || a.wHeight < 15 {
		return styles.CenterStyle.
			Width(a.wWidth).
			Height(a.wHeight).
			Render(
				styles.BorderStyle.
					Padding(1, 1).
					Foreground(lipgloss.Color("#ffffff")).
					BorderForeground(lipgloss.Color("#ff5f87")).
					Render("Window too small!"),
			)
	}

	// Show error if present
	if a.err != nil {
		return styles.ErrorStyle.Render(a.err.Error())
	}

	// Show loading if not ready
	if !a.ready {
		return styles.CenterStyle.
			Width(a.wWidth).
			Height(a.wHeight).
			Render(styles.MutedStyle.Render("Loading..."))
	}

	// Render current page
	currentPage := a.pages[a.currentPage]
	if currentPage == nil {
		return styles.ErrorStyle.Render("Page not found")
	}

	pageView := currentPage.View()

	// Create status bar if needed
	statusBar := ""
	if withHelp, ok := currentPage.(core.KeyMapHelp); ok {
		help := withHelp.Help()
		if help != nil {
			// Show short help
			shortcuts := help.ShortHelp()
			if len(shortcuts) > 0 {
				var helpParts []string
				for _, binding := range shortcuts {
					if binding.Help().Key != "" && binding.Help().Desc != "" {
						keyPart := styles.StatusStyle.Render(binding.Help().Key)
						actionPart := styles.ActionStyle.Render(binding.Help().Desc)
						helpParts = append(helpParts, keyPart+" "+actionPart)
					}
				}
				if len(helpParts) > 0 {
					// Join with proper spacing between key bindings
					statusText := ""
					for i, part := range helpParts {
						if i > 0 {
							statusText += "  " // Add two spaces between key bindings
						}
						statusText += part
					}
					statusBar = styles.BaseStyle.
						Width(a.width).
						PaddingLeft(1).
						Render(statusText)
				}
			}
		}
	}

	// Combine page view with status bar
	var components []string
	components = append(components, pageView)
	if statusBar != "" {
		components = append(components, statusBar)
	}

	baseView := lipgloss.JoinVertical(lipgloss.Top, components...)

	// Create layered view if there is a dialog
	if a.dialog.HasDialog() {
		// Create background layer with the base view
		baseLayer := lipgloss.NewLayer(baseView)

		// Get dialog layers
		dialogLayer := a.dialog.GetLayer()

		// Combine all layers
		allLayers := []*lipgloss.Layer{baseLayer}
		allLayers = append(allLayers, dialogLayer)

		// Create and render canvas
		canvas := lipgloss.NewCanvas(allLayers...)
		return canvas.Render()
	}

	return baseView
}

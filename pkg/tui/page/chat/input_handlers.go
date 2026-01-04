package chat

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/docker/cagent/pkg/tui/components/editor"
	"github.com/docker/cagent/pkg/tui/components/messages"
	"github.com/docker/cagent/pkg/tui/components/tool/editfile"
	"github.com/docker/cagent/pkg/tui/core/layout"
)

// handleKeyPress handles keyboard input events for the chat page.
// Returns the updated model and command, plus a bool indicating if the event was handled.
func (p *chatPage) handleKeyPress(msg tea.KeyPressMsg) (layout.Model, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, p.keyMap.Tab):
		if p.focusedPanel == PanelEditor && p.editor.AcceptSuggestion() {
			return p, nil, true
		}
		p.switchFocus()
		return p, nil, true

	case key.Matches(msg, p.keyMap.Cancel):
		cmd := p.cancelStream(true)
		return p, cmd, true

	case key.Matches(msg, p.keyMap.ExternalEditor):
		cmd := p.openExternalEditor()
		return p, cmd, true

	case key.Matches(msg, p.keyMap.ToggleSplitDiff):
		model, cmd := p.messages.Update(editfile.ToggleDiffViewMsg{})
		p.messages = model.(messages.Model)
		return p, cmd, true
	}

	// Route other keys to focused component
	switch p.focusedPanel {
	case PanelChat:
		model, cmd := p.messages.Update(msg)
		p.messages = model.(messages.Model)
		return p, cmd, true
	case PanelEditor:
		model, cmd := p.editor.Update(msg)
		p.editor = model.(editor.Editor)
		return p, cmd, true
	}

	return p, nil, false
}

// handleMouseClick handles mouse click events.
func (p *chatPage) handleMouseClick(msg tea.MouseClickMsg) (layout.Model, tea.Cmd) {
	if p.isOnResizeHandle(msg.X, msg.Y) {
		p.isDragging = true
		return p, nil
	}
	cmd := p.routeMouseEvent(msg, msg.Y)
	return p, cmd
}

// handleMouseMotion handles mouse motion events.
func (p *chatPage) handleMouseMotion(msg tea.MouseMotionMsg) (layout.Model, tea.Cmd) {
	if p.isDragging {
		cmd := p.handleResize(msg.Y)
		return p, cmd
	}
	p.isHoveringHandle = p.isOnResizeLine(msg.Y)
	cmd := p.routeMouseEvent(msg, msg.Y)
	return p, cmd
}

// handleMouseRelease handles mouse release events.
func (p *chatPage) handleMouseRelease(msg tea.MouseReleaseMsg) (layout.Model, tea.Cmd) {
	if p.isDragging {
		p.isDragging = false
		return p, nil
	}
	cmd := p.routeMouseEvent(msg, msg.Y)
	return p, cmd
}

// handleMouseWheel handles mouse wheel events.
func (p *chatPage) handleMouseWheel(msg tea.MouseWheelMsg) (layout.Model, tea.Cmd) {
	cmd := p.routeMouseEvent(msg, msg.Y)
	return p, cmd
}

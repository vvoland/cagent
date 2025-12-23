package core

import (
	tea "charm.land/bubbletea/v2"
)

// ScrollDirection represents the direction of scrolling
type ScrollDirection int

const (
	ScrollUp ScrollDirection = iota
	ScrollDown
	ScrollPageUp
	ScrollPageDown
	ScrollToTop
	ScrollToBottom
)

// KeyScrollMap maps common scroll keys to directions
var KeyScrollMap = map[string]ScrollDirection{
	"up":     ScrollUp,
	"k":      ScrollUp,
	"down":   ScrollDown,
	"j":      ScrollDown,
	"pgup":   ScrollPageUp,
	"pgdown": ScrollPageDown,
	"home":   ScrollToTop,
	"end":    ScrollToBottom,
}

// GetScrollDirection returns the scroll direction for a key press, or -1 if not a scroll key
func GetScrollDirection(msg tea.KeyPressMsg) (ScrollDirection, bool) {
	dir, ok := KeyScrollMap[msg.String()]
	return dir, ok
}

// IsQuitKey returns true if the key is ctrl+c
func IsQuitKey(msg tea.KeyPressMsg) bool {
	return msg.String() == "ctrl+c"
}

// IsEscapeKey returns true if the key is escape
func IsEscapeKey(msg tea.KeyPressMsg) bool {
	return msg.String() == "esc"
}

// MouseWheelDirection returns the scroll direction from a mouse wheel event
// Returns 1 for scroll down, -1 for scroll up, 0 for no scroll
func MouseWheelDirection(msg tea.MouseWheelMsg) int {
	buttonStr := msg.Button.String()
	switch buttonStr {
	case "wheelup":
		return -1
	case "wheeldown":
		return 1
	default:
		// Fallback to Y value for other wheel types
		if msg.Y < 0 {
			return -1
		} else if msg.Y > 0 {
			return 1
		}
		return 0
	}
}

// NavigationKeys are common keys used for navigation that components might want to handle
var NavigationKeys = []string{"up", "down", "left", "right", "k", "j", "h", "l", "enter", "esc"}

// IsNavigationKey returns true if the key is a common navigation key
func IsNavigationKey(msg tea.KeyPressMsg) bool {
	key := msg.String()
	for _, navKey := range NavigationKeys {
		if key == navKey {
			return true
		}
	}
	return false
}

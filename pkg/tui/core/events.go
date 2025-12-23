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

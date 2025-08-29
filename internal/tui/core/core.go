package core

import (
	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
)

// KeyMapHelp interface for components that provide help
type KeyMapHelp interface {
	Help() help.KeyMap
}

// simpleHelp implements help.KeyMap with simple key bindings
type simpleHelp struct {
	shortList []key.Binding
	fullList  [][]key.Binding
}

// NewSimpleHelp creates a new simple help system
func NewSimpleHelp(shortList []key.Binding, fullList [][]key.Binding) help.KeyMap {
	return &simpleHelp{
		shortList: shortList,
		fullList:  fullList,
	}
}

// FullHelp implements help.KeyMap
func (s *simpleHelp) FullHelp() [][]key.Binding {
	return s.fullList
}

// ShortHelp implements help.KeyMap
func (s *simpleHelp) ShortHelp() []key.Binding {
	return s.shortList
}

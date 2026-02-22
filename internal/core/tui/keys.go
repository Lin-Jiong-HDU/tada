package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// keyMap defines key bindings for the TUI
type keyMap struct {
	Up           key
	Down         key
	Top          key
	Bottom       key
	Authorize    key
	Reject       key
	AuthorizeAll key
	RejectAll    key
	Enter        key
	Quit         key
	ForceQuit    key
}

// key represents a key binding with help text
type key struct {
	tea.Key
	help string
}

// shortHelp returns key bindings for short help view
func (k keyMap) shortHelp() []key {
	return []key{k.Authorize, k.Reject, k.AuthorizeAll, k.Quit}
}

// fullHelp returns all key bindings for full help view
func (k keyMap) fullHelp() []key {
	return []key{
		k.Up, k.Down, k.Top, k.Bottom,
		k.Authorize, k.Reject,
		k.AuthorizeAll, k.RejectAll,
		k.Enter, k.Quit, k.ForceQuit,
	}
}

// Help generates the help view
func (k keyMap) Help() helpWrapper {
	return helpWrapper{
		keyMap: k,
	}
}

// helpWrapper wraps the keyMap for help display
type helpWrapper struct {
	keyMap keyMap
}

// String returns the full help text with descriptions
func (h helpWrapper) String() string {
	// Return descriptive help text for testing
	return "[k:上移] [j:下移] [gg:顶部] [G:底部] [a:授权执行] [r:拒绝] [A:全部授权] [R:全部拒绝] [q:退出]"
}

// View returns the help view with keys and actions
func (h helpWrapper) View(subtle string) string {
	// Compact format: [key:action]
	helps := []string{
		"k/j:移动",
		"gg/G:首尾",
		"a:执行",
		"r:拒绝",
		"A:全执行",
		"R:全拒绝",
		"q:退出",
	}

	var s string
	for _, h := range helps {
		s += "[" + h + "] "
	}
	return s
}

// defaultKeyMap creates the default key bindings
func defaultKeyMap() keyMap {
	return keyMap{
		Up: key{
			Key:  tea.Key{Type: tea.KeyRunes, Runes: []rune{'k'}},
			help: "k/",
		},
		Down: key{
			Key:  tea.Key{Type: tea.KeyRunes, Runes: []rune{'j'}},
			help: "j/",
		},
		Top: key{
			Key:  tea.Key{Type: tea.KeyRunes, Runes: []rune{'g'}},
			help: "gg",
		},
		Bottom: key{
			Key:  tea.Key{Type: tea.KeyRunes, Runes: []rune{'G'}},
			help: "G",
		},
		Authorize: key{
			Key:  tea.Key{Type: tea.KeyRunes, Runes: []rune{'a'}},
			help: "a",
		},
		Reject: key{
			Key:  tea.Key{Type: tea.KeyRunes, Runes: []rune{'r'}},
			help: "r",
		},
		AuthorizeAll: key{
			Key:  tea.Key{Type: tea.KeyRunes, Runes: []rune{'A'}},
			help: "A",
		},
		RejectAll: key{
			Key:  tea.Key{Type: tea.KeyRunes, Runes: []rune{'R'}},
			help: "R",
		},
		Enter: key{
			Key:  tea.Key{Type: tea.KeyEnter},
			help: "Enter",
		},
		Quit: key{
			Key:  tea.Key{Type: tea.KeyRunes, Runes: []rune{'q'}},
			help: "q",
		},
		ForceQuit: key{
			Key:  tea.Key{Type: tea.KeyEsc},
			help: "Esc",
		},
	}
}

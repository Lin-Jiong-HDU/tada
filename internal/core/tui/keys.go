package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// keyMap defines key bindings for the TUI
type keyMap struct {
	Up           key
	Down         key
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
		k.Up, k.Down,
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

// String returns the help text
func (h helpWrapper) String() string {
	var s string
	for _, k := range h.keyMap.fullHelp() {
		if k.help != "" {
			s += k.help + " "
		}
	}
	return s
}

// View returns the help view
func (h helpWrapper) View(subtle string) string {
	var s string
	for _, k := range h.keyMap.shortHelp() {
		s += "[" + k.help + "] "
	}
	return s
}

// defaultKeyMap creates the default key bindings
func defaultKeyMap() keyMap {
	return keyMap{
		Up: key{
			Key:  tea.Key{Type: tea.KeyRunes, Runes: []rune{'k'}},
			help: "↑/k",
		},
		Down: key{
			Key:  tea.Key{Type: tea.KeyRunes, Runes: []rune{'j'}},
			help: "↓/j",
		},
		Authorize: key{
			Key:  tea.Key{Type: tea.KeyRunes, Runes: []rune{'a'}},
			help: "授权选中",
		},
		Reject: key{
			Key:  tea.Key{Type: tea.KeyRunes, Runes: []rune{'r'}},
			help: "拒绝选中",
		},
		AuthorizeAll: key{
			Key:  tea.Key{Type: tea.KeyRunes, Runes: []rune{'A'}},
			help: "全部授权",
		},
		RejectAll: key{
			Key:  tea.Key{Type: tea.KeyRunes, Runes: []rune{'R'}},
			help: "全部拒绝",
		},
		Enter: key{
			Key:  tea.Key{Type: tea.KeyEnter},
			help: "查看详情",
		},
		Quit: key{
			Key:  tea.Key{Type: tea.KeyRunes, Runes: []rune{'q'}},
			help: "退出",
		},
		ForceQuit: key{
			Key:  tea.Key{Type: tea.KeyEsc},
			help: "强制退出",
		},
	}
}

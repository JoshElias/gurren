package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the key bindings for the TUI
type KeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Toggle key.Binding
	Quit   key.Binding
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/\u2191", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/\u2193", "down"),
		),
		Toggle: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "toggle"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// ShortHelp returns the short help text
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Toggle, k.Quit}
}

// FullHelp returns the full help text
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.Toggle, k.Quit},
	}
}

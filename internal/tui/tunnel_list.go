package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/JoshElias/gurren/internal/tunnel"
)

// TunnelItem represents a tunnel in the list
type TunnelItem struct {
	Name      string
	Host      string
	Status    tunnel.State
	Error     string
	Ephemeral bool
	Local     string
	Remote    string
}

// FilterValue implements list.Item for filtering
func (t TunnelItem) FilterValue() string {
	return t.Name
}

// Title implements list.DefaultItem (not used with custom delegate)
func (t TunnelItem) Title() string {
	return t.Name
}

// Description implements list.DefaultItem (not used with custom delegate)
func (t TunnelItem) Description() string {
	return fmt.Sprintf("%s -> %s", t.Local, t.Remote)
}

// TunnelDelegate is a custom item delegate for rendering tunnel items
type TunnelDelegate struct {
	ShowEphemeral bool
}

// NewTunnelDelegate creates a new tunnel delegate
func NewTunnelDelegate() TunnelDelegate {
	return TunnelDelegate{
		ShowEphemeral: true,
	}
}

// Height returns the height of each item
func (d TunnelDelegate) Height() int {
	return 1
}

// Spacing returns the spacing between items
func (d TunnelDelegate) Spacing() int {
	return 0
}

// Update handles item-level updates (not used)
func (d TunnelDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

// Render renders a single tunnel item
func (d TunnelDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	t, ok := item.(TunnelItem)
	if !ok {
		return
	}

	// Check if this item is selected
	isSelected := index == m.Index()

	// Status indicator
	statusIcon := StatusIcon(
		t.Status == tunnel.StateConnected,
		t.Status == tunnel.StateConnecting,
		t.Status == tunnel.StateError,
	)

	// Build the line
	var line strings.Builder

	// Cursor
	if isSelected {
		line.WriteString(cursorStyle.Render("> "))
	} else {
		line.WriteString("  ")
	}

	// Status icon
	line.WriteString(statusIcon)
	line.WriteString(" ")

	// Name with styling based on selection
	name := t.Name
	if t.Ephemeral && d.ShowEphemeral {
		name = name + " " + ephemeralStyle.Render(IconEphemeral)
	}

	if isSelected {
		line.WriteString(selectedStyle.Render(name))
	} else {
		line.WriteString(normalStyle.Render(name))
	}

	// Write to output
	fmt.Fprint(w, line.String())
}

// TunnelListPanel wraps a bubbles/list with panel styling
type TunnelListPanel struct {
	list   list.Model
	width  int
	height int
}

// NewTunnelListPanel creates a new tunnel list panel
func NewTunnelListPanel() TunnelListPanel {
	// Create list with custom delegate
	delegate := NewTunnelDelegate()

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()

	// Style the filter input
	l.FilterInput.PromptStyle = lipgloss.NewStyle().Foreground(colorBlue)
	l.FilterInput.TextStyle = lipgloss.NewStyle().Foreground(colorFg)
	l.FilterInput.Cursor.Style = lipgloss.NewStyle().Foreground(colorBlue)

	// Style empty state
	l.Styles.NoItems = mutedStyle

	return TunnelListPanel{
		list: l,
	}
}

// SetSize sets the panel dimensions
func (p *TunnelListPanel) SetSize(w, h int) {
	p.width = w
	p.height = h

	// Set list size (inside borders)
	contentW := w - 2
	contentH := h - 2
	if contentW < 0 {
		contentW = 0
	}
	if contentH < 0 {
		contentH = 0
	}
	p.list.SetSize(contentW, contentH)
}

// SetItems updates the tunnel list
func (p *TunnelListPanel) SetItems(items []TunnelItem) {
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}
	p.list.SetItems(listItems)
}

// Update handles list updates
func (p *TunnelListPanel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	return cmd
}

// SelectedItem returns the currently selected tunnel
func (p *TunnelListPanel) SelectedItem() *TunnelItem {
	item := p.list.SelectedItem()
	if item == nil {
		return nil
	}
	t, ok := item.(TunnelItem)
	if !ok {
		return nil
	}
	return &t
}

// SelectedIndex returns the currently selected index
func (p *TunnelListPanel) SelectedIndex() int {
	return p.list.Index()
}

// Filtering returns true if the list is in filtering mode
func (p *TunnelListPanel) Filtering() bool {
	return p.list.FilterState() == list.Filtering
}

// View renders the list panel
func (p TunnelListPanel) View() string {
	// Wrap list view in panel style
	return panelStyle.
		Width(p.width).
		Height(p.height).
		Render(p.list.View())
}

// Items returns all items in the list
func (p *TunnelListPanel) Items() []TunnelItem {
	items := p.list.Items()
	tunnels := make([]TunnelItem, 0, len(items))
	for _, item := range items {
		if t, ok := item.(TunnelItem); ok {
			tunnels = append(tunnels, t)
		}
	}
	return tunnels
}

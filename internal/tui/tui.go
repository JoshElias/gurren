package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/JoshElias/gurren/internal/daemon"
	"github.com/JoshElias/gurren/internal/tunnel"
)

// Model is the main TUI model
type Model struct {
	// Components
	listPanel    TunnelListPanel
	detailsPanel DetailsPanel
	statusBar    StatusBar

	// State
	keys   KeyMap
	client *daemon.Client
	width  int
	height int
	err    error
}

// Messages

// tunnelsLoadedMsg is sent when tunnels are loaded from the daemon
type tunnelsLoadedMsg struct {
	tunnels []TunnelItem
}

// tunnelStatusChangedMsg is sent when a tunnel status changes
type tunnelStatusChangedMsg struct {
	name   string
	status tunnel.State
	err    string
}

// errorMsg is sent when an error occurs
type errorMsg struct {
	err error
}

// notificationMsg wraps a daemon notification
type notificationMsg daemon.Notification

// New creates a new TUI model
func New(client *daemon.Client) Model {
	keys := DefaultKeyMap()
	return Model{
		listPanel:    NewTunnelListPanel(),
		detailsPanel: NewDetailsPanel(),
		statusBar:    NewStatusBar(keys),
		keys:         keys,
		client:       client,
	}
}

// Init initializes the TUI
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadTunnels(),
		m.listenForNotifications(),
	)
}

// loadTunnels loads tunnels from the daemon
func (m Model) loadTunnels() tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.TunnelList()
		if err != nil {
			return errorMsg{err}
		}

		tunnels := make([]TunnelItem, len(result.Tunnels))
		for i, t := range result.Tunnels {
			tunnels[i] = TunnelItem{
				Name:      t.Name,
				Host:      t.Config.Host,
				Status:    t.Status,
				Error:     t.Error,
				Ephemeral: t.Ephemeral,
				Local:     t.Config.Local,
				Remote:    t.Config.Remote,
			}
		}

		return tunnelsLoadedMsg{tunnels}
	}
}

// listenForNotifications listens for status change notifications from the daemon
func (m Model) listenForNotifications() tea.Cmd {
	return func() tea.Msg {
		notif, ok := <-m.client.Notifications()
		if !ok {
			return nil
		}
		return notificationMsg(notif)
	}
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()
		return m, nil

	case tea.KeyMsg:
		// If list is filtering, let it handle all keys
		if m.listPanel.Filtering() {
			cmd := m.listPanel.Update(msg)
			return m, cmd
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Up), key.Matches(msg, m.keys.Down):
			cmd := m.listPanel.Update(msg)
			return m, cmd

		case key.Matches(msg, m.keys.Filter):
			// Start filtering
			cmd := m.listPanel.Update(msg)
			return m, cmd

		case key.Matches(msg, m.keys.Toggle):
			if selected := m.listPanel.SelectedItem(); selected != nil {
				return m, m.toggleTunnel(selected.Name)
			}
			return m, nil
		}

	case tunnelsLoadedMsg:
		m.listPanel.SetItems(msg.tunnels)
		return m, nil

	case tunnelStatusChangedMsg:
		// Update the tunnel in the list
		items := m.listPanel.Items()
		for i := range items {
			if items[i].Name == msg.name {
				items[i].Status = msg.status
				items[i].Error = msg.err

				// Show toast on error
				if msg.status == tunnel.StateError && msg.err != "" {
					m.statusBar.SetToast(msg.err, ToastError)
					cmds = append(cmds, HideToastCmd())
				}
				break
			}
		}
		m.listPanel.SetItems(items)
		return m, tea.Batch(cmds...)

	case notificationMsg:
		// Parse the notification and convert to status change
		if msg.Method == daemon.MethodStatusChanged {
			var params daemon.StatusChangedParams
			if err := json.Unmarshal(msg.Params, &params); err == nil {
				// Update and continue listening
				listenCmd := m.listenForNotifications()
				newModel, updateCmd := m.Update(tunnelStatusChangedMsg{
					name:   params.Name,
					status: params.Status,
					err:    params.Error,
				})
				return newModel, tea.Batch(listenCmd, updateCmd)
			}
		}
		return m, m.listenForNotifications()

	case errorMsg:
		m.statusBar.SetToast(msg.err.Error(), ToastError)
		return m, HideToastCmd()

	case hideToastMsg:
		m.statusBar.ClearToast()
		return m, nil
	}

	return m, nil
}

// updateLayout recalculates component sizes based on terminal dimensions
func (m *Model) updateLayout() {
	// Reserve height for status bar (1 line + padding)
	statusBarHeight := 1
	contentHeight := m.height - statusBarHeight

	// Calculate panel widths
	// List panel: 33% of width, min 25, max 50
	listWidth := m.width / 3
	if listWidth < 25 {
		listWidth = 25
	}
	if listWidth > 50 {
		listWidth = 50
	}

	// Ensure we don't exceed terminal width
	if listWidth > m.width {
		listWidth = m.width
	}

	detailsWidth := m.width - listWidth
	if detailsWidth < 0 {
		detailsWidth = 0
	}

	// Update component sizes
	m.listPanel.SetSize(listWidth, contentHeight)
	m.detailsPanel.SetSize(detailsWidth, contentHeight)
	m.statusBar.SetWidth(m.width)
}

// toggleTunnel toggles the connection status of a tunnel
func (m Model) toggleTunnel(name string) tea.Cmd {
	return func() tea.Msg {
		// Find current status
		var currentStatus tunnel.State
		for _, t := range m.listPanel.Items() {
			if t.Name == name {
				currentStatus = t.Status
				break
			}
		}

		if currentStatus.IsActive() {
			// Stop tunnel
			if err := m.client.TunnelStop(name); err != nil {
				return errorMsg{err}
			}
		} else {
			// Start tunnel
			if _, err := m.client.TunnelStart(name); err != nil {
				return errorMsg{err}
			}
		}

		return nil
	}
}

// View renders the TUI
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	// Check for empty state
	if len(m.listPanel.Items()) == 0 && !m.listPanel.Filtering() {
		return m.renderEmptyState()
	}

	// Render panels
	listView := m.listPanel.View()
	detailsView := m.detailsPanel.View(m.listPanel.SelectedItem())

	// Join panels horizontally
	panels := lipgloss.JoinHorizontal(lipgloss.Top, listView, detailsView)

	// Add status bar
	statusBar := m.statusBar.View()

	// Join vertically
	return lipgloss.JoinVertical(lipgloss.Left, panels, statusBar)
}

// renderEmptyState renders the empty state when no tunnels are configured
func (m Model) renderEmptyState() string {
	// Get config path for help text
	configPath := "~/.config/gurren/config.toml"
	if home, err := os.UserHomeDir(); err == nil {
		configPath = filepath.Join(home, ".config", "gurren", "config.toml")
	}

	var b strings.Builder

	// Title
	b.WriteString(emptyStateTitleStyle.Render("No tunnels configured"))
	b.WriteString("\n\n")

	// Instructions
	b.WriteString(mutedStyle.Render("Create a config file at:"))
	b.WriteString("\n")
	b.WriteString(emptyStateCodeStyle.Render(configPath))
	b.WriteString("\n\n")

	// Example
	b.WriteString(mutedStyle.Render("Example:"))
	b.WriteString("\n")
	example := `[[tunnels]]
name = "production-db"
host = "user@bastion.example.com"
remote = "db.internal:5432"
local = "localhost:5432"`
	b.WriteString(emptyStateCodeStyle.Render(example))
	b.WriteString("\n\n")

	// Or use ad-hoc
	b.WriteString(mutedStyle.Render("Or use ad-hoc connections:"))
	b.WriteString("\n")
	b.WriteString(emptyStateCodeStyle.Render("gurren connect --host user@bastion --remote db:5432 --local :5432"))

	content := b.String()

	// Center in available space
	contentHeight := m.height - 1 // Reserve for status bar
	centered := lipgloss.Place(m.width, contentHeight, lipgloss.Center, lipgloss.Center, content)

	// Add status bar
	return lipgloss.JoinVertical(lipgloss.Left, centered, m.statusBar.View())
}

// Run starts the TUI
func Run(client *daemon.Client) error {
	// Subscribe to notifications
	if err := client.Subscribe(); err != nil {
		return fmt.Errorf("failed to subscribe to notifications: %w", err)
	}

	p := tea.NewProgram(
		New(client),
		tea.WithAltScreen(),
	)

	_, err := p.Run()
	return err
}

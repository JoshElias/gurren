package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/JoshElias/gurren/internal/daemon"
	"github.com/JoshElias/gurren/internal/tunnel"
)

// TunnelItem represents a tunnel in the list
type TunnelItem struct {
	Name      string
	Status    tunnel.State
	Error     string
	Ephemeral bool
	Local     string
	Remote    string
}

// Model is the main TUI model
type Model struct {
	tunnels []TunnelItem
	cursor  int
	keys    KeyMap
	client  *daemon.Client
	toast   *Toast
	width   int
	height  int
	err     error
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
	return Model{
		keys:   DefaultKeyMap(),
		client: client,
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
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case key.Matches(msg, m.keys.Down):
			if m.cursor < len(m.tunnels)-1 {
				m.cursor++
			}
			return m, nil

		case key.Matches(msg, m.keys.Toggle):
			if len(m.tunnels) > 0 {
				return m, m.toggleTunnel(m.tunnels[m.cursor].Name)
			}
			return m, nil
		}

	case tunnelsLoadedMsg:
		m.tunnels = msg.tunnels
		return m, nil

	case tunnelStatusChangedMsg:
		for i := range m.tunnels {
			if m.tunnels[i].Name == msg.name {
				m.tunnels[i].Status = msg.status
				m.tunnels[i].Error = msg.err

				// Show toast on error
				if msg.status == tunnel.StateError && msg.err != "" {
					m.toast = NewToast(msg.err)
					return m, hideToastCmd()
				}
				break
			}
		}
		return m, nil

	case notificationMsg:
		// Parse the notification and convert to status change
		if msg.Method == daemon.MethodStatusChanged {
			var params daemon.StatusChangedParams
			if err := json.Unmarshal(msg.Params, &params); err == nil {
				// Update inline and continue listening
				cmd := m.listenForNotifications()
				newModel, _ := m.Update(tunnelStatusChangedMsg{
					name:   params.Name,
					status: params.Status,
					err:    params.Error,
				})
				return newModel, cmd
			}
		}
		return m, m.listenForNotifications()

	case errorMsg:
		m.toast = NewToast(msg.err.Error())
		return m, hideToastCmd()

	case hideToastMsg:
		m.toast = nil
		return m, nil
	}

	return m, nil
}

// toggleTunnel toggles the connection status of a tunnel
func (m Model) toggleTunnel(name string) tea.Cmd {
	return func() tea.Msg {
		// Find current status
		var currentStatus tunnel.State
		for _, t := range m.tunnels {
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
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("Gurren"))
	b.WriteString("\n\n")

	// Tunnel list
	if len(m.tunnels) == 0 {
		b.WriteString(mutedStyle.Render("No tunnels configured"))
		b.WriteString("\n")
	} else {
		for i, t := range m.tunnels {
			// Cursor
			if i == m.cursor {
				b.WriteString(cursorStyle.String())
			} else {
				b.WriteString(noCursorStyle.String())
			}
			b.WriteString(" ")

			// Status indicator
			switch t.Status {
			case tunnel.StateConnected:
				b.WriteString(statusConnected.String())
			case tunnel.StateConnecting:
				b.WriteString(statusConnecting.String())
			case tunnel.StateError:
				b.WriteString(statusError.String())
			default:
				b.WriteString(statusDisconnected.String())
			}
			b.WriteString(" ")

			// Name and details
			if i == m.cursor {
				b.WriteString(selectedStyle.Render(t.Name))
			} else {
				b.WriteString(normalStyle.Render(t.Name))
			}
			b.WriteString("\n")

			// Details on second line
			details := fmt.Sprintf("  %s -> %s", t.Local, t.Remote)
			b.WriteString(mutedStyle.Render(details))
			b.WriteString("\n")
		}
	}

	// Help bar
	helpText := fmt.Sprintf(
		"%s navigate  %s toggle  %s quit",
		helpKeyStyle.Render("j/k"),
		helpKeyStyle.Render("enter"),
		helpKeyStyle.Render("q"),
	)
	b.WriteString(helpStyle.Render(helpText))

	// Compose the main view
	mainView := b.String()

	// Overlay toast in top-right if visible
	if m.toast != nil && m.width > 0 {
		toastView := m.toast.View(40)
		if toastView != "" {
			// Calculate position for top-right
			toastWidth := lipgloss.Width(toastView)
			padding := m.width - toastWidth - 2
			if padding > 0 {
				toastView = strings.Repeat(" ", padding) + toastView
			}
			// Prepend toast to main view
			mainView = toastView + "\n" + mainView
		}
	}

	return mainView
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

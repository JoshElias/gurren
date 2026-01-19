package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/JoshElias/gurren/internal/tunnel"
)

// DetailsPanel renders the right panel showing selected tunnel details
type DetailsPanel struct {
	width  int
	height int
}

// NewDetailsPanel creates a new details panel
func NewDetailsPanel() DetailsPanel {
	return DetailsPanel{}
}

// SetSize sets the panel dimensions
func (d *DetailsPanel) SetSize(w, h int) {
	d.width = w
	d.height = h
}

// View renders the details panel for the given tunnel
func (d DetailsPanel) View(item *TunnelItem) string {
	// Content area dimensions (inside border)
	contentWidth := d.width - 2 // account for left/right borders
	if contentWidth < 0 {
		contentWidth = 0
	}
	contentHeight := d.height - 2 // account for top/bottom borders
	if contentHeight < 0 {
		contentHeight = 0
	}

	var content string
	if item == nil {
		// Empty state
		content = d.renderEmpty(contentWidth, contentHeight)
	} else {
		// Render tunnel details
		content = d.renderDetails(item, contentWidth, contentHeight)
	}

	// Wrap in panel style
	return panelStyle.
		Width(d.width).
		Height(d.height).
		Render(content)
}

// renderEmpty renders the empty state for the details panel
func (d DetailsPanel) renderEmpty(width, height int) string {
	msg := mutedStyle.Render("No tunnel selected")
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, msg)
}

// renderDetails renders the tunnel details
func (d DetailsPanel) renderDetails(item *TunnelItem, width, height int) string {
	var lines []string

	// Parse host into user, hostname, port
	user, host, port := parseHost(item.Host)

	// Name
	lines = append(lines, d.renderRow(IconName, "Name", item.Name))
	lines = append(lines, "")

	// Status with colored indicator
	statusIcon := StatusIcon(
		item.Status == tunnel.StateConnected,
		item.Status == tunnel.StateConnecting,
		item.Status == tunnel.StateError,
	)
	statusText := StatusText(
		item.Status == tunnel.StateConnected,
		item.Status == tunnel.StateConnecting,
		item.Status == tunnel.StateError,
	)
	lines = append(lines, d.renderRow(IconStatus, "Status", statusIcon+" "+statusText))

	// Error message if present
	if item.Error != "" {
		lines = append(lines, "")
		errorText := statusErrorStyle.Render(item.Error)
		lines = append(lines, d.renderRowValue("", "Error", errorText))
	}

	// Ephemeral indicator
	if item.Ephemeral {
		lines = append(lines, "")
		ephText := ephemeralStyle.Render(IconEphemeral + " Ephemeral (ad-hoc)")
		lines = append(lines, ephText)
	}

	lines = append(lines, "")

	// Connection details
	if user != "" {
		lines = append(lines, d.renderRow(IconUser, "User", user))
	}
	lines = append(lines, d.renderRow(IconHost, "Host", host))
	if port != "" && port != "22" {
		lines = append(lines, d.renderRow(IconPort, "Port", port))
	}

	lines = append(lines, "")

	// Tunnel endpoints
	lines = append(lines, d.renderRow(IconLocal, "Local", item.Local))
	lines = append(lines, d.renderRow(IconRemote, "Remote", item.Remote))

	content := strings.Join(lines, "\n")

	// Pad to fill height
	lineCount := strings.Count(content, "\n") + 1
	if lineCount < height {
		content += strings.Repeat("\n", height-lineCount)
	}

	return content
}

// renderRow renders a labeled row with icon
func (d DetailsPanel) renderRow(icon, label, value string) string {
	iconPart := mutedStyle.Render(icon)
	labelPart := labelStyle.Render(label)
	valuePart := valueStyle.Render(value)
	return iconPart + " " + labelPart + valuePart
}

// renderRowValue renders a labeled row with pre-styled value
func (d DetailsPanel) renderRowValue(icon, label, styledValue string) string {
	iconPart := mutedStyle.Render(icon)
	labelPart := labelStyle.Render(label)
	if icon == "" {
		return "  " + labelPart + styledValue
	}
	return iconPart + " " + labelPart + styledValue
}

// parseHost parses a host string like "user@hostname:port" into components
func parseHost(host string) (user, hostname, port string) {
	// Default port
	port = "22"

	// Check for user@
	if idx := strings.Index(host, "@"); idx != -1 {
		user = host[:idx]
		host = host[idx+1:]
	}

	// Check for :port
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		hostname = host[:idx]
		port = host[idx+1:]
	} else {
		hostname = host
	}

	return user, hostname, port
}

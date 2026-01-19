package tui

import "github.com/charmbracelet/lipgloss"

// OneDark color palette
var (
	colorBg        = lipgloss.Color("#282c34") // Main background
	colorBgLight   = lipgloss.Color("#2c313a") // Slightly lighter bg
	colorBorder    = lipgloss.Color("#3b3f4c") // Border color
	colorFg        = lipgloss.Color("#abb2bf") // Main text
	colorBlue      = lipgloss.Color("#61afef") // Primary/selected
	colorGreen     = lipgloss.Color("#98c379") // Connected
	colorOrange    = lipgloss.Color("#d19a66") // Connecting
	colorRed       = lipgloss.Color("#e86671") // Error
	colorGrey      = lipgloss.Color("#7f848e") // Muted (brightened for readability)
	colorLightGrey = lipgloss.Color("#9da5b4") // Secondary text (brightened for readability)
	colorCyan      = lipgloss.Color("#56b6c2") // Accent
	colorPurple    = lipgloss.Color("#c678dd") // Ephemeral tunnels
)

// Nerd Font icons
const (
	IconConnected    = "\uf00c" //  (checkmark)
	IconDisconnected = "\uf10c" //  (circle outline)
	IconConnecting   = "\uf110" //  (spinner)
	IconError        = "\uf00d" //  (x mark)
	IconTunnel       = "󰛳"      // Panel title - network
	IconDetails      = ""       // Panel title - info
	IconUser         = ""       // User field
	IconHost         = "󰒋"      // Host field
	IconPort         = "󰙜"      // Port field
	IconLocal        = "󰌘"      // Local field
	IconRemote       = "󰒍"      // Remote field
	IconStatus       = ""       // Status field
	IconEphemeral    = ""       // Ephemeral indicator
	IconName         = ""       // Name field
)

// Panel styles
var (
	// Base panel style with rounded border
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder)

	// Focused panel has blue border
	focusedPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorBlue)

	// Panel title style (embedded in border)
	panelTitleStyle = lipgloss.NewStyle().
			Foreground(colorBlue).
			Bold(true).
			Padding(0, 1)
)

// Text styles
var (
	// Normal text
	normalStyle = lipgloss.NewStyle().
			Foreground(colorFg)

	// Muted text
	mutedStyle = lipgloss.NewStyle().
			Foreground(colorGrey)

	// Secondary text
	secondaryStyle = lipgloss.NewStyle().
			Foreground(colorLightGrey)

	// Selected/highlighted text
	selectedStyle = lipgloss.NewStyle().
			Foreground(colorBlue).
			Bold(true)

	// Label style (for details panel)
	labelStyle = lipgloss.NewStyle().
			Foreground(colorGrey).
			Width(16)

	// Value style (for details panel)
	valueStyle = lipgloss.NewStyle().
			Foreground(colorFg)
)

// Status styles
var (
	statusConnectedStyle = lipgloss.NewStyle().
				Foreground(colorGreen)

	statusDisconnectedStyle = lipgloss.NewStyle().
				Foreground(colorGrey)

	statusConnectingStyle = lipgloss.NewStyle().
				Foreground(colorOrange)

	statusErrorStyle = lipgloss.NewStyle().
				Foreground(colorRed)
)

// List item styles
var (
	// Cursor style
	cursorStyle = lipgloss.NewStyle().
			Foreground(colorBlue).
			Bold(true)

	// Selected list item background
	selectedItemStyle = lipgloss.NewStyle().
				Background(colorBgLight).
				Foreground(colorBlue)

	// Normal list item
	normalItemStyle = lipgloss.NewStyle().
			Foreground(colorFg)

	// Ephemeral tunnel indicator
	ephemeralStyle = lipgloss.NewStyle().
			Foreground(colorPurple)
)

// Status bar styles
var (
	statusBarStyle = lipgloss.NewStyle().
			Foreground(colorGrey)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(colorLightGrey)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(colorGrey)

	toastErrorStyle = lipgloss.NewStyle().
			Foreground(colorRed)

	toastSuccessStyle = lipgloss.NewStyle().
				Foreground(colorGreen)
)

// Empty state style
var (
	emptyStateStyle = lipgloss.NewStyle().
			Foreground(colorGrey).
			Align(lipgloss.Center)

	emptyStateTitleStyle = lipgloss.NewStyle().
				Foreground(colorLightGrey).
				Bold(true).
				MarginBottom(1)

	emptyStateCodeStyle = lipgloss.NewStyle().
				Foreground(colorCyan)
)

// Helper functions

// StatusIcon returns the appropriate icon for a tunnel state
func StatusIcon(connected, connecting, hasError bool) string {
	switch {
	case hasError:
		return statusErrorStyle.Render(IconError)
	case connecting:
		return statusConnectingStyle.Render(IconConnecting)
	case connected:
		return statusConnectedStyle.Render(IconConnected)
	default:
		return statusDisconnectedStyle.Render(IconDisconnected)
	}
}

// StatusText returns styled status text
func StatusText(connected, connecting, hasError bool) string {
	switch {
	case hasError:
		return statusErrorStyle.Render("Error")
	case connecting:
		return statusConnectingStyle.Render("Connecting")
	case connected:
		return statusConnectedStyle.Render("Connected")
	default:
		return statusDisconnectedStyle.Render("Disconnected")
	}
}

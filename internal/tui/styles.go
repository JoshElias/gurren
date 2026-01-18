package tui

import "github.com/charmbracelet/lipgloss"

// Colors
var (
	colorPrimary   = lipgloss.Color("205") // Pink/magenta
	colorSecondary = lipgloss.Color("240") // Gray
	colorSuccess   = lipgloss.Color("42")  // Green
	colorWarning   = lipgloss.Color("214") // Yellow/orange
	colorError     = lipgloss.Color("196") // Red
	colorMuted     = lipgloss.Color("241") // Muted gray
)

// Styles
var (
	// Title style
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			MarginBottom(1)

	// Tunnel list item styles
	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	mutedStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	// Status indicator styles
	statusConnected = lipgloss.NewStyle().
			Foreground(colorSuccess).
			SetString("\u25cf") // ●

	statusDisconnected = lipgloss.NewStyle().
				Foreground(colorMuted).
				SetString("\u25cb") // ○

	statusConnecting = lipgloss.NewStyle().
				Foreground(colorWarning).
				SetString("\u25d0") // ◐

	statusError = lipgloss.NewStyle().
			Foreground(colorError).
			SetString("\u25cf") // ●

	// Cursor
	cursorStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			SetString(">")

	noCursorStyle = lipgloss.NewStyle().
			SetString(" ")

	// Help bar
	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			MarginTop(1)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(colorSecondary)

	// Toast styles
	toastStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorError).
			Padding(0, 1).
			Foreground(colorError)

	toastIconStyle = lipgloss.NewStyle().
			Foreground(colorError).
			SetString("\u2717 ") // ✗
)

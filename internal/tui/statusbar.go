package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const toastDuration = 5 * time.Second

// ToastType represents the type of toast message
type ToastType int

const (
	ToastError ToastType = iota
	ToastSuccess
	ToastInfo
)

// StatusBar combines help text on the left and toast messages on the right
type StatusBar struct {
	width   int
	keys    KeyMap
	toast   string
	toastTy ToastType
}

// NewStatusBar creates a new status bar
func NewStatusBar(keys KeyMap) StatusBar {
	return StatusBar{
		keys: keys,
	}
}

// SetWidth sets the status bar width
func (s *StatusBar) SetWidth(w int) {
	s.width = w
}

// SetToast sets a toast message
func (s *StatusBar) SetToast(msg string, ty ToastType) {
	s.toast = msg
	s.toastTy = ty
}

// ClearToast clears the toast message
func (s *StatusBar) ClearToast() {
	s.toast = ""
}

// HasToast returns true if there is a toast message
func (s *StatusBar) HasToast() bool {
	return s.toast != ""
}

// hideToastMsg is sent when the toast should be hidden
type hideToastMsg struct{}

// HideToastCmd returns a command that hides the toast after the duration
func HideToastCmd() tea.Cmd {
	return tea.Tick(toastDuration, func(time.Time) tea.Msg {
		return hideToastMsg{}
	})
}

// View renders the status bar
func (s StatusBar) View() string {
	if s.width == 0 {
		return ""
	}

	// Build help text from key bindings
	helpParts := []string{}
	for _, binding := range s.keys.ShortHelp() {
		help := binding.Help()
		if help.Key != "" && help.Desc != "" {
			part := helpKeyStyle.Render(help.Key) + " " + helpDescStyle.Render(help.Desc)
			helpParts = append(helpParts, part)
		}
	}
	helpText := strings.Join(helpParts, "  ")

	// If no toast, just render help text
	if s.toast == "" {
		return statusBarStyle.Width(s.width).Render(helpText)
	}

	// Render toast with appropriate style
	var toastText string
	switch s.toastTy {
	case ToastError:
		toastText = toastErrorStyle.Render(IconError + " " + s.toast)
	case ToastSuccess:
		toastText = toastSuccessStyle.Render(IconConnected + " " + s.toast)
	default:
		toastText = secondaryStyle.Render(s.toast)
	}

	// Calculate widths for layout
	helpWidth := lipgloss.Width(helpText)
	toastWidth := lipgloss.Width(toastText)
	availableWidth := s.width

	// If both fit, render side by side with space between
	if helpWidth+toastWidth+4 <= availableWidth {
		gap := availableWidth - helpWidth - toastWidth
		return helpText + strings.Repeat(" ", gap) + toastText
	}

	// If toast is too long, truncate help or show only toast
	if toastWidth < availableWidth {
		return lipgloss.NewStyle().Width(availableWidth).Align(lipgloss.Right).Render(toastText)
	}

	// Toast is very long, truncate it
	maxLen := availableWidth - 3
	if maxLen > 0 && len(s.toast) > maxLen {
		truncated := s.toast[:maxLen] + "..."
		switch s.toastTy {
		case ToastError:
			toastText = toastErrorStyle.Render(IconError + " " + truncated)
		case ToastSuccess:
			toastText = toastSuccessStyle.Render(IconConnected + " " + truncated)
		default:
			toastText = secondaryStyle.Render(truncated)
		}
	}

	return toastText
}

package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const toastDuration = 5 * time.Second

// Toast represents an error notification popup
type Toast struct {
	message string
	visible bool
}

// hideToastMsg is sent when the toast should be hidden
type hideToastMsg struct{}

// NewToast creates a new toast with a message
func NewToast(message string) *Toast {
	return &Toast{
		message: message,
		visible: true,
	}
}

// View renders the toast
func (t *Toast) View(maxWidth int) string {
	if !t.visible || t.message == "" {
		return ""
	}

	content := toastIconStyle.String() + t.message
	return toastStyle.MaxWidth(maxWidth).Render(content)
}

// hideToastCmd returns a command that hides the toast after the duration
func hideToastCmd() tea.Cmd {
	return tea.Tick(toastDuration, func(time.Time) tea.Msg {
		return hideToastMsg{}
	})
}

package tui

import (
	"github.com/Lin-Jiong-HDU/tada/internal/core/queue"
	tea "github.com/charmbracelet/bubbletea"
)

// AuthorizeResultMsg is sent when authorization completes
type AuthorizeResultMsg struct {
	TaskID  string
	Success bool
}

// RejectResultMsg is sent when rejection completes
type RejectResultMsg struct {
	TaskID  string
	Success bool
}

// TickMsg is sent for UI updates
type TickMsg struct{}

// TasksLoadedMsg is sent when tasks are loaded
type TasksLoadedMsg struct {
	Tasks []*queue.Task
}

// Model is the interface for the TUI model
type Model interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (tea.Model, tea.Cmd)
	View() string
}

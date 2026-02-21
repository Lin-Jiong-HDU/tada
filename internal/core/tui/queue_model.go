package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Lin-Jiong-HDU/tada/internal/core/queue"
)

// model is the Bubble Tea model for the queue TUI
type model struct {
	tasks       []*queue.Task
	cursor      int
	selected    map[string]struct{}
	keys        keyMap
	showingHelp bool
	onAuthorize func(string) tea.Cmd
	onReject    func(string) tea.Cmd
}

// NewModel creates a new queue UI model
func NewModel(tasks []*queue.Task) Model {
	return model{
		tasks:       tasks,
		cursor:      0,
		selected:    make(map[string]struct{}),
		keys:        defaultKeyMap(),
		showingHelp: false,
		onAuthorize: defaultAuthorizeHandler,
		onReject:    defaultRejectHandler,
	}
}

func defaultAuthorizeHandler(taskID string) tea.Cmd {
	return func() tea.Msg {
		return AuthorizeResultMsg{TaskID: taskID, Success: true}
	}
}

func defaultRejectHandler(taskID string) tea.Cmd {
	return func() tea.Msg {
		return RejectResultMsg{TaskID: taskID, Success: true}
	}
}

// Init initializes the model
func (m model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case AuthorizeResultMsg:
		if msg.Success {
			// Update task status
			for i, task := range m.tasks {
				if task.ID == msg.TaskID {
					m.tasks[i].Status = queue.TaskStatusApproved
					break
				}
			}
		}
		return m, nil

	case RejectResultMsg:
		if msg.Success {
			for i, task := range m.tasks {
				if task.ID == msg.TaskID {
					m.tasks[i].Status = queue.TaskStatusRejected
					break
				}
			}
		}
		return m, nil
	}

	return m, nil
}

func (m model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle quit
	if msg.String() == "q" || msg.String() == "ctrl+c" || msg.Type == tea.KeyEsc {
		return m, tea.Quit
	}

	// Handle help toggle
	if msg.Type == tea.KeyEnter {
		m.showingHelp = !m.showingHelp
		return m, nil
	}

	// Handle navigation
	switch msg.String() {
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "j", "down":
		if m.cursor < len(m.tasks)-1 {
			m.cursor++
		}
	}

	// Handle actions
	switch msg.String() {
	case "a":
		if len(m.tasks) > 0 {
			task := m.tasks[m.cursor]
			return m, m.onAuthorize(task.ID)
		}
	case "r":
		if len(m.tasks) > 0 {
			task := m.tasks[m.cursor]
			return m, m.onReject(task.ID)
		}
	case "A":
		var cmds []tea.Cmd
		for _, task := range m.tasks {
			if task.Status == queue.TaskStatusPending {
				cmds = append(cmds, m.onAuthorize(task.ID))
			}
		}
		if len(cmds) > 0 {
			return m, tea.Batch(cmds...)
		}
	case "R":
		var cmds []tea.Cmd
		for _, task := range m.tasks {
			if task.Status == queue.TaskStatusPending {
				cmds = append(cmds, m.onReject(task.ID))
			}
		}
		if len(cmds) > 0 {
			return m, tea.Batch(cmds...)
		}
	}

	return m, nil
}

// View renders the UI
func (m model) View() string {
	if m.showingHelp {
		return m.renderHelp()
	}
	return m.renderQueue()
}

func (m model) renderQueue() string {
	var s string

	// Header
	s += titleStyle.Render(" tada 任务队列 ") + "\n\n"

	if len(m.tasks) == 0 {
		s += subtleStyle.Render("没有待授权任务") + "\n"
		return s
	}

	// Group by session
	grouped := m.groupTasksBySession()

	for sessionID, tasks := range grouped {
		s += fmt.Sprintf(" 会话: %s \n", sessionID)

		for _, task := range tasks {
			cursor := " "
			if m.getCursorForTask(task.ID) == m.cursor {
				cursor = ">"
			}

			// Status indicator
			status := getStatusIndicator(task.Status)

			// Command string
			cmdStr := task.Command.Cmd
			if len(task.Command.Args) > 0 {
				cmdStr += " " + fmt.Sprint(task.Command.Args)
			}

			// Truncate if too long
			if len(cmdStr) > 50 {
				cmdStr = cmdStr[:47] + "..."
			}

			s += fmt.Sprintf("%s [%s] %s\n", cursor, status, cmdStr)

			if task.CheckResult != nil && task.CheckResult.Warning != "" {
				s += subtleStyle.Render("     警告: "+task.CheckResult.Warning) + "\n"
			}
		}
		s += "\n"
	}

	// Footer
	s += m.renderFooter()

	return s
}

func (m model) renderHelp() string {
	return helpStyle.Render(`
按 q 返回队列
`) + "\n"
}

func (m model) renderFooter() string {
	return "\n" + m.keys.Help().View("")
}

func (m model) groupTasksBySession() map[string][]*queue.Task {
	grouped := make(map[string][]*queue.Task)
	currentIdx := 0

	for _, task := range m.tasks {
		// Only show pending tasks
		if task.Status == queue.TaskStatusPending {
			grouped[task.SessionID] = append(grouped[task.SessionID], task)
		}
		if task.ID == m.tasks[m.cursor].ID {
			m.cursor = currentIdx
		}
		currentIdx++
	}

	return grouped
}

func (m model) getCursorForTask(taskID string) int {
	for i, task := range m.tasks {
		if task.ID == taskID {
			return i
		}
	}
	return 0
}

func getStatusIndicator(status queue.TaskStatus) string {
	switch status {
	case queue.TaskStatusPending:
		return " "
	case queue.TaskStatusApproved:
		return "✓"
	case queue.TaskStatusRejected:
		return "✗"
	case queue.TaskStatusExecuting:
		return "⋯"
	case queue.TaskStatusCompleted:
		return "✓"
	case queue.TaskStatusFailed:
		return "!"
	default:
		return "?"
	}
}

// Styles
var (
	titleStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	subtleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

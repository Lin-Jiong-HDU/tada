package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Lin-Jiong-HDU/tada/internal/core/queue"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TaskReloadFunc is a callback to reload a task from the queue
type TaskReloadFunc func(taskID string) *queue.Task

// model is the Bubble Tea model for the queue TUI
type model struct {
	tasks          []*queue.Task
	cursor         int
	selected       map[string]struct{}
	keys           keyMap
	showingHelp    bool
	onAuthorize    func(string) tea.Cmd
	onReject       func(string) tea.Cmd
	taskReloadFunc TaskReloadFunc
	pendingG       bool // Tracks if 'g' was pressed for 'gg' command
	width          int
	height         int
}

// NewModel creates a new queue UI model
func NewModel(tasks []*queue.Task) Model {
	return NewModelWithOptions(tasks, nil, nil, nil)
}

// NewModelWithOptions creates a new queue UI model with custom authorize/reject handlers
func NewModelWithOptions(tasks []*queue.Task, onAuthorize, onReject func(string) tea.Cmd, taskReloadFunc TaskReloadFunc) Model {
	if onAuthorize == nil {
		onAuthorize = defaultAuthorizeHandler
	}
	if onReject == nil {
		onReject = defaultRejectHandler
	}

	return model{
		tasks:          tasks,
		cursor:         0,
		selected:       make(map[string]struct{}),
		keys:           defaultKeyMap(),
		showingHelp:    false,
		onAuthorize:    onAuthorize,
		onReject:       onReject,
		taskReloadFunc: taskReloadFunc,
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
	return tea.Batch(
		tea.WindowSize(),
	)
}

// Update handles messages
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case AuthorizeResultMsg:
		if msg.Success {
			// Update task status to executing (will be updated by background executor)
			for i, task := range m.tasks {
				if task.ID == msg.TaskID {
					// Create a copy with updated status
					updatedTask := *task
					updatedTask.Status = queue.TaskStatusExecuting
					m.tasks[i] = &updatedTask

					// Start a ticker to check for status updates
					return m, tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
						return StatusCheckMsg{TaskID: msg.TaskID}
					})
				}
			}
		}
		return m, nil

	case StatusCheckMsg:
		// Reload task from queue to get fresh status
		if m.taskReloadFunc != nil {
			if freshTask := m.taskReloadFunc(msg.TaskID); freshTask != nil {
				// Update the task in our list with fresh data
				for i, task := range m.tasks {
					if task.ID == msg.TaskID {
						m.tasks[i] = freshTask
						// If still executing, schedule another check
						if freshTask.Status == queue.TaskStatusExecuting {
							return m, tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
								return StatusCheckMsg{TaskID: msg.TaskID}
							})
						}
						break
					}
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
		m.pendingG = false
		if m.cursor > 0 {
			m.cursor--
		}
	case "j", "down":
		m.pendingG = false
		if m.cursor < len(m.tasks)-1 {
			m.cursor++
		}
	case "g":
		// Handle vim-style gg to go to top
		if m.pendingG {
			m.cursor = 0
			m.pendingG = false
		} else {
			m.pendingG = true
		}
	case "G":
		m.pendingG = false
		// Go to bottom
		if len(m.tasks) > 0 {
			m.cursor = len(m.tasks) - 1
		}
	default:
		m.pendingG = false
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

	// Content area
	var content string
	if len(m.tasks) == 0 {
		content = subtleStyle.Render("没有待授权任务")
	} else {
		// Group by session
		grouped := m.groupTasksBySession()

		for sessionID, tasks := range grouped {
			content += fmt.Sprintf(" 会话: %s \n", sessionID)

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
					cmdStr += " " + strings.Join(task.Command.Args, " ")
				}

				// Truncate if too long
				if len(cmdStr) > 50 {
					cmdStr = cmdStr[:47] + "..."
				}

				content += fmt.Sprintf("%s [%s] %s\n", cursor, status, cmdStr)

				if task.CheckResult != nil && task.CheckResult.Warning != "" {
					content += subtleStyle.Render("     警告: "+task.CheckResult.Warning) + "\n"
				}
			}
			content += "\n"
		}
	}

	// Get footer
	footer := m.renderFooter()

	// Calculate lines to push footer to bottom
	// Count lines in header (2), content, and footer
	headerLines := 2
	contentLines := countLines(content)
	footerLines := countLines(footer)

	totalLines := headerLines + contentLines + footerLines

	// If we have window height, add padding to push footer to bottom
	if m.height > 0 {
		paddingNeeded := m.height - totalLines
		if paddingNeeded > 0 {
			// Add empty lines to push footer to bottom
			for i := 0; i < paddingNeeded; i++ {
				content += "\n"
			}
		}
	}

	// Combine content with footer
	s += content + footer

	return s
}

// countLines counts the number of lines in a string
func countLines(s string) int {
	if s == "" {
		return 0
	}
	count := 0
	for _, ch := range s {
		if ch == '\n' {
			count++
		}
	}
	// If string doesn't end with newline, count the last line
	if len(s) > 0 && s[len(s)-1] != '\n' {
		count++
	}
	return count
}

func (m model) renderHelp() string {
	return helpStyle.Render(`
按 q 退出程序
`) + "\n"
}

func (m model) renderFooter() string {
	// Get help text
	helpText := m.keys.Help().View("")

	// Style the help as a status bar with border
	statusBar := statusBarStyle.Render(helpText)

	return "\n" + statusBar + "\n"
}

func (m model) groupTasksBySession() map[string][]*queue.Task {
	grouped := make(map[string][]*queue.Task)

	for _, task := range m.tasks {
		// Show pending and executing tasks
		if task.Status == queue.TaskStatusPending || task.Status == queue.TaskStatusExecuting {
			grouped[task.SessionID] = append(grouped[task.SessionID], task)
		}
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
	titleStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	subtleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	helpStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("235")).
			Padding(0, 1).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("241")).
			MarginTop(1)
)

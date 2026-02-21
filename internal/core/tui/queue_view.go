package tui

import (
	"fmt"

	"github.com/Lin-Jiong-HDU/tada/internal/core/queue"
	"github.com/charmbracelet/lipgloss"
)

// Renderer handles TUI rendering
type Renderer struct {
	width  int
	height int
	style  *StyleConfig
}

// StyleConfig defines visual styles
type StyleConfig struct {
	TitleColor    lipgloss.Color
	SubtleColor   lipgloss.Color
	ErrorColor    lipgloss.Color
	SuccessColor  lipgloss.Color
	WarningColor  lipgloss.Color
	SelectedColor lipgloss.Color
	BorderColor   lipgloss.Color
}

// DefaultStyleConfig returns the default style configuration
func DefaultStyleConfig() *StyleConfig {
	return &StyleConfig{
		TitleColor:    lipgloss.Color("10"),  // Green
		SubtleColor:   lipgloss.Color("241"), // Grey
		ErrorColor:    lipgloss.Color("9"),   // Red
		SuccessColor:  lipgloss.Color("10"),  // Green
		WarningColor:  lipgloss.Color("11"),  // Yellow
		SelectedColor: lipgloss.Color("12"),  // Blue
		BorderColor:   lipgloss.Color("8"),   // Dark grey
	}
}

// NewRenderer creates a new TUI renderer
func NewRenderer(width, height int) *Renderer {
	return &Renderer{
		width:  width,
		height: height,
		style:  DefaultStyleConfig(),
	}
}

// Render renders the full TUI view
func (r *Renderer) Render(mdl *model) string {
	return r.renderHeader() + "\n" + r.renderTasks(mdl) + "\n" + r.renderFooter(mdl)
}

func (r *Renderer) renderHeader() string {
	title := lipgloss.NewStyle().
		Foreground(r.style.TitleColor).
		Bold(true).
		Render("tada 任务队列")

	border := lipgloss.NewStyle().
		Foreground(r.style.BorderColor).
		Render("──────────────────────────────────────────────────────────────")

	return title + "\n" + border
}

func (r *Renderer) renderTasks(mdl *model) string {
	if len(mdl.tasks) == 0 {
		return r.renderEmptyState()
	}

	// Group by session
	grouped := mdl.groupTasksBySession()

	if len(grouped) == 0 {
		return r.renderEmptyState()
	}

	var result string

	for sessionID, tasks := range grouped {
		result += r.renderSessionHeader(sessionID)

		for _, task := range tasks {
			result += r.renderTask(mdl, task)
		}

		result += "\n"
	}

	return result
}

func (r *Renderer) renderEmptyState() string {
	return lipgloss.NewStyle().
		Foreground(r.style.SubtleColor).
		Render("\n  没有待授权任务\n")
}

func (r *Renderer) renderSessionHeader(sessionID string) string {
	style := lipgloss.NewStyle().
		Foreground(r.style.SelectedColor).
		Bold(true)

	return fmt.Sprintf("\n  %s\n", style.Render("会话: "+sessionID))
}

func (r *Renderer) renderTask(mdl *model, task *queue.Task) string {
	// Determine if selected
	isSelected := mdl.cursor == r.getTaskIndex(mdl, task.ID)
	cursor := " "
	if isSelected {
		cursor = ">"
	}

	// Status indicator
	status := r.renderStatus(task.Status)

	// Command string
	cmdStr := r.renderCommand(task)

	// Warning
	warning := ""
	if task.CheckResult != nil && task.CheckResult.Warning != "" {
		warning = lipgloss.NewStyle().
			Foreground(r.style.WarningColor).
			Render("     警告: " + task.CheckResult.Warning + "\n")
	}

	// Build task line
	content := fmt.Sprintf("[%s] %s", status, cmdStr)
	if warning != "" {
		content = fmt.Sprintf("[%s] %s\n%s", status, cmdStr, warning)
	}

	return fmt.Sprintf("  %s %s\n", cursor, content)
}

func (r *Renderer) renderStatus(status queue.TaskStatus) string {
	var symbol string
	var color lipgloss.Color

	switch status {
	case queue.TaskStatusPending:
		symbol = " "
		color = r.style.SubtleColor
	case queue.TaskStatusApproved:
		symbol = "✓"
		color = r.style.SuccessColor
	case queue.TaskStatusRejected:
		symbol = "✗"
		color = r.style.ErrorColor
	case queue.TaskStatusExecuting:
		symbol = "⋯"
		color = r.style.WarningColor
	case queue.TaskStatusCompleted:
		symbol = "✓"
		color = r.style.SuccessColor
	case queue.TaskStatusFailed:
		symbol = "!"
		color = r.style.ErrorColor
	default:
		symbol = "?"
		color = r.style.SubtleColor
	}

	return lipgloss.NewStyle().Foreground(color).Render(symbol)
}

func (r *Renderer) renderCommand(task *queue.Task) string {
	cmdStr := task.Command.Cmd
	if len(task.Command.Args) > 0 {
		cmdStr += " " + fmt.Sprint(task.Command.Args)
	}

	// Truncate if too long
	maxLen := 60
	if len(cmdStr) > maxLen {
		cmdStr = cmdStr[:maxLen-3] + "..."
	}

	return lipgloss.NewStyle().
		Foreground(r.style.TitleColor).
		Render(cmdStr)
}

func (r *Renderer) renderFooter(mdl *model) string {
	help := mdl.keys.Help().View("")

	style := lipgloss.NewStyle().
		Foreground(r.style.SubtleColor)

	return "\n" + style.Render(help)
}

func (r *Renderer) getTaskIndex(mdl *model, taskID string) int {
	for i, task := range mdl.tasks {
		if task.ID == taskID {
			return i
		}
	}
	return -1
}

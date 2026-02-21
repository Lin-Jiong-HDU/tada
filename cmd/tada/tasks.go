package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Lin-Jiong-HDU/tada/internal/core"
	"github.com/Lin-Jiong-HDU/tada/internal/core/execution"
	"github.com/Lin-Jiong-HDU/tada/internal/core/queue"
	"github.com/Lin-Jiong-HDU/tada/internal/core/tui"
	"github.com/Lin-Jiong-HDU/tada/internal/storage"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// getTasksCommand returns the tasks command
func getTasksCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "tasks",
		Short: "管理待授权命令队列",
		Long: `打开 TUI 界面管理需要授权的命令。

查看、授权或拒绝待授权的异步命令。`,
		RunE: runTasks,
	}
}

func runTasks(cmd *cobra.Command, args []string) error {
	// Get sessions directory
	configDir, err := storage.GetConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	sessionsDir := filepath.Join(configDir, storage.SessionDirName)

	// Load all queues and tasks
	queues, allTasks, err := loadAllQueues(sessionsDir)
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	// Filter pending tasks
	var pendingTasks []*queue.Task
	for _, task := range allTasks {
		if task.Status == queue.TaskStatusPending {
			pendingTasks = append(pendingTasks, task)
		}
	}

	// Create handlers that persist to the appropriate queue and execute
	onAuthorize := func(taskID string) tea.Cmd {
		return func() tea.Msg {
			q := findQueueForTask(queues, taskID)
			if q != nil {
				// Approve the task
				if err := q.ApproveTask(taskID); err != nil {
					return tui.AuthorizeResultMsg{TaskID: taskID, Success: false}
				}

				// Execute the task immediately
				ctx := context.Background()
				executor := core.NewExecutor(30 * time.Second)
				taskExecutor := execution.NewTaskExecutor(q, executor)

				go func() {
					_ = taskExecutor.ExecuteTask(ctx, taskID)
				}()
			}
			return tui.AuthorizeResultMsg{TaskID: taskID, Success: true}
		}
	}

	onReject := func(taskID string) tea.Cmd {
		return func() tea.Msg {
			q := findQueueForTask(queues, taskID)
			if q != nil {
				if err := q.RejectTask(taskID); err != nil {
					return tui.RejectResultMsg{TaskID: taskID, Success: false}
				}
			}
			return tui.RejectResultMsg{TaskID: taskID, Success: true}
		}
	}

	// Create TUI model with persistence handlers
	model := tui.NewModelWithOptions(pendingTasks, onAuthorize, onReject)

	// Run TUI
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}

// loadAllQueues loads all queue managers and their tasks
func loadAllQueues(sessionsDir string) (map[string]*queue.Manager, []*queue.Task, error) {
	queues := make(map[string]*queue.Manager)
	var allTasks []*queue.Task

	// Read all session directories
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return queues, allTasks, nil
		}
		return nil, nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			sessionDir := filepath.Join(sessionsDir, entry.Name())
			queueFile := filepath.Join(sessionDir, "queue.json")

			// Create queue manager
			q := queue.NewQueue(queueFile, entry.Name())
			tasks := q.GetAllTasks()

			queues[entry.Name()] = q
			allTasks = append(allTasks, tasks...)
		}
	}

	return queues, allTasks, nil
}

// findQueueForTask finds the queue manager that contains the given task
func findQueueForTask(queues map[string]*queue.Manager, taskID string) *queue.Manager {
	for _, q := range queues {
		tasks := q.GetAllTasks()
		for _, task := range tasks {
			if task.ID == taskID {
				return q
			}
		}
	}
	return nil
}

// loadAllTasks loads all tasks (deprecated, use loadAllQueues)
func loadAllTasks(sessionsDir string) ([]*queue.Task, error) {
	_, allTasks, err := loadAllQueues(sessionsDir)
	return allTasks, err
}

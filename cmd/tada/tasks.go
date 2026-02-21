package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/Lin-Jiong-HDU/tada/internal/core/queue"
	"github.com/Lin-Jiong-HDU/tada/internal/core/tui"
	"github.com/Lin-Jiong-HDU/tada/internal/storage"
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

	// Load all queue files
	allTasks, err := loadAllTasks(sessionsDir)
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

	// Create TUI model
	model := tui.NewModel(pendingTasks)

	// Run TUI
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}

func loadAllTasks(sessionsDir string) ([]*queue.Task, error) {
	var allTasks []*queue.Task

	// Read all session directories
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return allTasks, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			sessionDir := filepath.Join(sessionsDir, entry.Name())
			queueFile := filepath.Join(sessionDir, "queue.json")

			// Load queue file
			q := queue.NewQueue(queueFile, entry.Name())
			tasks := q.GetAllTasks()
			allTasks = append(allTasks, tasks...)
		}
	}

	return allTasks, nil
}

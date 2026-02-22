package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/Lin-Jiong-HDU/tada/internal/core"
	"github.com/Lin-Jiong-HDU/tada/internal/core/execution"
	"github.com/Lin-Jiong-HDU/tada/internal/core/queue"
	"github.com/Lin-Jiong-HDU/tada/internal/storage"
	"github.com/spf13/cobra"
)

// getRunCommand returns the run command
func getRunCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "执行所有已批准的任务",
		Long: `执行队列中所有已批准但尚未执行的任务。

适用于批量执行之前在 TUI 中授权的任务。`,
		RunE: runApprovedTasks,
	}
}

func runApprovedTasks(cmd *cobra.Command, args []string) error {
	// Get sessions directory
	configDir, err := storage.GetConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	sessionsDir := filepath.Join(configDir, storage.SessionDirName)

	// Load all queues
	queues, _, err := loadAllQueues(sessionsDir)
	if err != nil {
		return fmt.Errorf("failed to load queues: %w", err)
	}

	if len(queues) == 0 {
		fmt.Println("没有找到任务队列")
		return nil
	}

	// Create executor
	executor := core.NewExecutor(30 * time.Second)
	ctx := context.Background()

	totalExecuted := 0
	totalFailed := 0

	// Execute approved tasks from each queue
	for sessionID, q := range queues {
		tasks := q.GetAllTasks()
		var approvedCount int

		for _, task := range tasks {
			if task.Status == queue.TaskStatusApproved {
				approvedCount++
			}
		}

		if approvedCount == 0 {
			continue
		}

		fmt.Printf("会话 %s: 执行 %d 个已批准任务...\n", sessionID, approvedCount)

		taskExecutor := execution.NewTaskExecutor(q, executor)
		results, err := taskExecutor.ExecuteAllApproved(ctx)

		executed := len(results)
		totalExecuted += executed

		if err != nil {
			fmt.Printf("  部分任务执行失败: %v\n", err)
			totalFailed++
		}

		// Show results
		tasks = q.GetAllTasks()
		for _, task := range tasks {
			if task.Status == queue.TaskStatusCompleted {
				fmt.Printf("  ✓ [%s] %s\n", task.ID[:8], task.Command.Cmd)
			} else if task.Status == queue.TaskStatusFailed {
				fmt.Printf("  ✗ [%s] %s\n", task.ID[:8], task.Command.Cmd)
				if task.Result != nil && task.Result.Error != "" {
					fmt.Printf("    错误: %s\n", task.Result.Error)
				}
			}
		}
	}

	if totalExecuted == 0 {
		fmt.Println("没有已批准的任务需要执行")
		fmt.Println("提示: 使用 'tada tasks' 查看并授权任务")
	} else {
		fmt.Printf("\n执行完成: %d 个任务", totalExecuted)
		if totalFailed > 0 {
			fmt.Printf(" (%d 个失败)", totalFailed)
		}
		fmt.Println()
	}

	return nil
}

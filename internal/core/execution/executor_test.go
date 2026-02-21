package execution

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
	"github.com/Lin-Jiong-HDU/tada/internal/core"
	"github.com/Lin-Jiong-HDU/tada/internal/core/queue"
	"github.com/Lin-Jiong-HDU/tada/internal/core/security"
)

func TestTaskExecutor_ExecuteApprovedTask(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "tada-executor-test-*")
	defer os.RemoveAll(tmpDir)

	queueFile := filepath.Join(tmpDir, "queue.json")
	q := queue.NewQueue(queueFile, "test-session")

	// Create and approve a task
	cmd := ai.Command{Cmd: "echo", Args: []string{"hello"}, IsAsync: true}
	check := &security.CheckResult{Allowed: true, RequiresAuth: true}
	task, _ := q.AddTask(cmd, check)
	q.ApproveTask(task.ID)

	// Create executor
	executor := core.NewExecutor(5 * time.Second)
	taskExecutor := NewTaskExecutor(q, executor)

	// Execute the task
	err := taskExecutor.ExecuteTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("ExecuteTask failed: %v", err)
	}

	// Verify task status
	tasks := q.GetAllTasks()
	if len(tasks) != 1 {
		t.Fatal("Expected 1 task")
	}

	if tasks[0].Status != queue.TaskStatusCompleted {
		t.Errorf("Expected completed status, got %s", tasks[0].Status)
	}

	if tasks[0].Result == nil {
		t.Error("Expected result to be set")
	} else if tasks[0].Result.Output != "hello" {
		t.Errorf("Expected output 'hello', got '%s'", tasks[0].Result.Output)
	}
}

func TestTaskExecutor_ExecuteAllApproved(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "tada-executor-test-*")
	defer os.RemoveAll(tmpDir)

	queueFile := filepath.Join(tmpDir, "queue.json")
	q := queue.NewQueue(queueFile, "test-session")

	// Create multiple tasks
	cmd1 := ai.Command{Cmd: "echo", Args: []string{"one"}}
	cmd2 := ai.Command{Cmd: "echo", Args: []string{"two"}}
	check := &security.CheckResult{Allowed: true, RequiresAuth: true}

	task1, _ := q.AddTask(cmd1, check)
	task2, _ := q.AddTask(cmd2, check)

	// Approve both
	q.ApproveTask(task1.ID)
	q.ApproveTask(task2.ID)

	// Execute all approved
	executor := core.NewExecutor(5 * time.Second)
	taskExecutor := NewTaskExecutor(q, executor)

	results, err := taskExecutor.ExecuteAllApproved(context.Background())
	if err != nil {
		t.Fatalf("ExecuteAllApproved failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Verify all completed
	tasks := q.GetAllTasks()
	completed := 0
	for _, task := range tasks {
		if task.Status == queue.TaskStatusCompleted {
			completed++
		}
	}
	if completed != 2 {
		t.Errorf("Expected 2 completed tasks, got %d", completed)
	}
}

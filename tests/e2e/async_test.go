package e2e

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
	"github.com/Lin-Jiong-HDU/tada/internal/core"
	"github.com/Lin-Jiong-HDU/tada/internal/core/execution"
	"github.com/Lin-Jiong-HDU/tada/internal/core/queue"
	"github.com/Lin-Jiong-HDU/tada/internal/core/security"
	"github.com/Lin-Jiong-HDU/tada/internal/storage"
)

func TestAsync_FullWorkflow(t *testing.T) {
	if os.Getenv("TADA_INTEGRATION_TEST") == "" {
		t.Skip("Set TADA_INTEGRATION_TEST=1 to run E2E tests")
	}

	// Setup temp directory
	oldHome := os.Getenv("HOME")
	tmpDir, err := os.MkdirTemp("", "tada-async-e2e-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Initialize config and session
	_, err = storage.InitConfig()
	if err != nil {
		t.Fatal(err)
	}

	session, err := storage.InitSession()
	if err != nil {
		t.Fatal(err)
	}

	// Create queue
	queueFile := filepath.Join(tmpDir, ".tada", storage.SessionDirName, session.ID, "queue.json")
	q := queue.NewQueue(queueFile, session.ID)

	// Test async syntax parsing
	isAsync := core.ParseAsyncSyntax("create folder &")
	if !isAsync {
		t.Error("Expected async syntax to be detected")
	}

	stripped := core.StripAsyncSyntax("create folder &")
	if stripped != "create folder" {
		t.Errorf("Expected 'create folder', got '%s'", stripped)
	}

	// Test queue workflow
	cmd := ai.Command{Cmd: "echo", Args: []string{"test"}, IsAsync: true}
	check := &security.CheckResult{Allowed: true, RequiresAuth: true}

	task, err := q.AddTask(cmd, check)
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	if task.Status != queue.TaskStatusPending {
		t.Errorf("Expected pending, got %s", task.Status)
	}

	// Approve and execute
	err = q.ApproveTask(task.ID)
	if err != nil {
		t.Fatalf("ApproveTask failed: %v", err)
	}

	executor := core.NewExecutor(5 * time.Second)
	taskExecutor := execution.NewTaskExecutor(q, executor)

	err = taskExecutor.ExecuteTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("ExecuteTask failed: %v", err)
	}

	// Verify completion
	tasks := q.GetAllTasks()
	if len(tasks) != 1 {
		t.Fatal("Expected 1 task")
	}

	if tasks[0].Status != queue.TaskStatusCompleted {
		t.Errorf("Expected completed, got %s", tasks[0].Status)
	}

	if tasks[0].Result == nil {
		t.Fatal("Expected result to be set")
	}

	t.Log("Async workflow E2E test passed!")
}

func TestAsync_SyntaxEdgeCases(t *testing.T) {
	if os.Getenv("TADA_INTEGRATION_TEST") == "" {
		t.Skip("Set TADA_INTEGRATION_TEST=1 to run E2E tests")
	}

	tests := []struct {
		name     string
		input    string
		wantAsync bool
		expected string
	}{
		{"multiple ampersands", "command && &", true, "command &&"},
		{"ampersand in middle", "command & stuff", false, "command & stuff"},
		{"just ampersand", "&", true, ""},
		{"no spaces", "command&", true, "command"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isAsync := core.ParseAsyncSyntax(tt.input)
			if isAsync != tt.wantAsync {
				t.Errorf("ParseAsyncSyntax() = %v, want %v", isAsync, tt.wantAsync)
			}

			stripped := core.StripAsyncSyntax(tt.input)
			if stripped != tt.expected {
				t.Errorf("StripAsyncSyntax() = %q, want %q", stripped, tt.expected)
			}
		})
	}
}

func TestAsync_MultiTaskExecution(t *testing.T) {
	if os.Getenv("TADA_INTEGRATION_TEST") == "" {
		t.Skip("Set TADA_INTEGRATION_TEST=1 to run E2E tests")
	}

	// Setup temp directory
	oldHome := os.Getenv("HOME")
	tmpDir, err := os.MkdirTemp("", "tada-async-multi-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Initialize config and session
	_, err = storage.InitConfig()
	if err != nil {
		t.Fatal(err)
	}

	session, err := storage.InitSession()
	if err != nil {
		t.Fatal(err)
	}

	// Create queue
	queueFile := filepath.Join(tmpDir, ".tada", storage.SessionDirName, session.ID, "queue.json")
	q := queue.NewQueue(queueFile, session.ID)

	// Add multiple tasks
	check := &security.CheckResult{Allowed: true, RequiresAuth: true}

	cmd1 := ai.Command{Cmd: "echo", Args: []string{"one"}}
	cmd2 := ai.Command{Cmd: "echo", Args: []string{"two"}}
	cmd3 := ai.Command{Cmd: "echo", Args: []string{"three"}}

	task1, _ := q.AddTask(cmd1, check)
	task2, _ := q.AddTask(cmd2, check)
	task3, _ := q.AddTask(cmd3, check)

	// Approve all
	q.ApproveTask(task1.ID)
	q.ApproveTask(task2.ID)
	q.ApproveTask(task3.ID)

	// Execute all approved
	executor := core.NewExecutor(5 * time.Second)
	taskExecutor := execution.NewTaskExecutor(q, executor)

	ctx := context.Background()
	results, err := taskExecutor.ExecuteAllApproved(ctx)
	if err != nil {
		t.Fatalf("ExecuteAllApproved failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// Verify all completed
	tasks := q.GetAllTasks()
	completed := 0
	for _, task := range tasks {
		if task.Status == queue.TaskStatusCompleted {
			completed++
		}
	}

	if completed != 3 {
		t.Errorf("Expected 3 completed tasks, got %d", completed)
	}

	t.Log("Multi-task async execution E2E test passed!")
}

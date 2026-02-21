package queue

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
	"github.com/Lin-Jiong-HDU/tada/internal/core/security"
)

func TestIntegration_FullQueueWorkflow(t *testing.T) {
	// Setup
	tmpDir, _ := os.MkdirTemp("", "tada-integration-*")
	defer os.RemoveAll(tmpDir)

	queueFile := filepath.Join(tmpDir, "queue.json")
	q := NewQueue(queueFile, "test-session-123")

	// Step 1: Add tasks
	cmd1 := ai.Command{Cmd: "rm", Args: []string{"-rf", "/tmp/test"}, IsAsync: true}
	check1 := &security.CheckResult{
		Allowed:      true,
		RequiresAuth: true,
		Warning:      "Dangerous command",
		Reason:       "In dangerous list",
	}

	task1, err := q.AddTask(cmd1, check1)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	if task1.Status != TaskStatusPending {
		t.Errorf("Expected pending status, got %s", task1.Status)
	}

	// Step 2: Verify persistence by creating new queue instance
	q2 := NewQueue(queueFile, "test-session-123")
	tasks := q2.GetAllTasks()

	if len(tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(tasks))
	}

	// Step 3: Approve task
	if err := q2.ApproveTask(task1.ID); err != nil {
		t.Fatalf("Failed to approve: %v", err)
	}

	// Step 4: Verify status change
	tasks = q2.GetAllTasks()
	if tasks[0].Status != TaskStatusApproved {
		t.Errorf("Expected approved status, got %s", tasks[0].Status)
	}

	// Step 5: Mark as executing
	if err := q2.MarkExecuting(task1.ID); err != nil {
		t.Fatalf("Failed to mark executing: %v", err)
	}

	// Step 6: Set result
	result := &ExecutionResult{
		ExitCode: 0,
		Output:   "success",
	}
	if err := q2.SetTaskResult(task1.ID, result); err != nil {
		t.Fatalf("Failed to set result: %v", err)
	}

	// Step 7: Verify final state
	tasks = q2.GetAllTasks()
	if tasks[0].Status != TaskStatusCompleted {
		t.Errorf("Expected completed status, got %s", tasks[0].Status)
	}

	if tasks[0].Result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", tasks[0].Result.ExitCode)
	}
}

func TestIntegration_MultipleSessions(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "tada-integration-*")
	defer os.RemoveAll(tmpDir)

	queueFile := filepath.Join(tmpDir, "queue.json")

	// Create queue for session 1
	q1 := NewQueue(queueFile, "session-1")
	cmd1 := ai.Command{Cmd: "rm", Args: []string{"-rf", "/tmp/test1"}}
	check1 := &security.CheckResult{Allowed: true, RequiresAuth: true}
	q1.AddTask(cmd1, check1)

	// Create queue for session 2 (same file, different session)
	q2 := NewQueue(queueFile, "session-2")
	cmd2 := ai.Command{Cmd: "dd", Args: []string{"if=/dev/zero", "of=file"}}
	check2 := &security.CheckResult{Allowed: true, RequiresAuth: true}
	q2.AddTask(cmd2, check2)

	// Verify both sessions' tasks
	session1Tasks := q2.GetTasksBySession("session-1")
	session2Tasks := q2.GetTasksBySession("session-2")

	if len(session1Tasks) != 1 {
		t.Errorf("Expected 1 task for session-1, got %d", len(session1Tasks))
	}

	if len(session2Tasks) != 1 {
		t.Errorf("Expected 1 task for session-2, got %d", len(session2Tasks))
	}

	// Total tasks should be 2
	allTasks := q2.GetAllTasks()
	if len(allTasks) != 2 {
		t.Errorf("Expected 2 total tasks, got %d", len(allTasks))
	}
}

func TestIntegration_StatusTransitionValidation(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "tada-integration-*")
	defer os.RemoveAll(tmpDir)

	queueFile := filepath.Join(tmpDir, "queue.json")
	q := NewQueue(queueFile, "test-session")

	cmd := ai.Command{Cmd: "rm", Args: []string{"-rf", "/tmp/test"}}
	check := &security.CheckResult{Allowed: true, RequiresAuth: true}

	task, _ := q.AddTask(cmd, check)

	// Invalid transition: pending -> completed
	if err := q.SetTaskResult(task.ID, &ExecutionResult{ExitCode: 0}); err == nil {
		t.Error("Expected error for invalid status transition")
	}

	// Valid: pending -> approved
	if err := q.ApproveTask(task.ID); err != nil {
		t.Errorf("Expected approve to succeed: %v", err)
	}

	// Invalid: approved -> approved (no change)
	if err := q.ApproveTask(task.ID); err == nil {
		t.Error("Expected error for duplicate approval")
	}
}

func TestIntegration_ConcurrentAccess(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "tada-integration-*")
	defer os.RemoveAll(tmpDir)

	queueFile := filepath.Join(tmpDir, "queue.json")
	q := NewQueue(queueFile, "test-session")

	// Add multiple tasks concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			cmd := ai.Command{Cmd: "echo", Args: []string{"test", fmt.Sprint(n)}}
			check := &security.CheckResult{Allowed: true, RequiresAuth: true}
			q.AddTask(cmd, check)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all tasks were added
	tasks := q.GetAllTasks()
	if len(tasks) != 10 {
		t.Errorf("Expected 10 tasks, got %d", len(tasks))
	}
}

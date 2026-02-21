package queue

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
	"github.com/Lin-Jiong-HDU/tada/internal/core/security"
)

func TestQueue_AddTask(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tada-queue-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	queueFile := filepath.Join(tmpDir, "queue.json")
	q := NewQueue(queueFile, "session-123")

	cmd := ai.Command{Cmd: "rm", Args: []string{"-rf", "/tmp/test"}}
	check := &security.CheckResult{Allowed: true, RequiresAuth: true}

	task, err := q.AddTask(cmd, check)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	if task.Status != TaskStatusPending {
		t.Errorf("Expected status pending, got %s", task.Status)
	}

	// Verify task was persisted
	tasks := q.GetAllTasks()
	if len(tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(tasks))
	}
}

func TestQueue_ApproveTask(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tada-queue-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	queueFile := filepath.Join(tmpDir, "queue.json")
	q := NewQueue(queueFile, "session-123")

	cmd := ai.Command{Cmd: "rm", Args: []string{"-rf", "/tmp/test"}}
	check := &security.CheckResult{Allowed: true, RequiresAuth: true}

	task, _ := q.AddTask(cmd, check)

	// Approve the task
	if err := q.ApproveTask(task.ID); err != nil {
		t.Fatalf("Failed to approve: %v", err)
	}

	// Reload to verify persistence
	tasks := q.GetAllTasks()
	if len(tasks) != 1 {
		t.Fatal("Expected 1 task")
	}

	if tasks[0].Status != TaskStatusApproved {
		t.Errorf("Expected status approved, got %s", tasks[0].Status)
	}
}

func TestQueue_RejectTask(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tada-queue-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	queueFile := filepath.Join(tmpDir, "queue.json")
	q := NewQueue(queueFile, "session-123")

	cmd := ai.Command{Cmd: "rm", Args: []string{"-rf", "/tmp/test"}}
	check := &security.CheckResult{Allowed: true, RequiresAuth: true}

	task, _ := q.AddTask(cmd, check)

	if err := q.RejectTask(task.ID); err != nil {
		t.Fatalf("Failed to reject: %v", err)
	}

	tasks := q.GetAllTasks()
	if tasks[0].Status != TaskStatusRejected {
		t.Errorf("Expected status rejected, got %s", tasks[0].Status)
	}
}

func TestQueue_GetPendingTasks(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tada-queue-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	queueFile := filepath.Join(tmpDir, "queue.json")
	q := NewQueue(queueFile, "session-123")

	// Add multiple tasks
	cmd := ai.Command{Cmd: "rm", Args: []string{"-rf", "/tmp/test"}}
	check := &security.CheckResult{Allowed: true, RequiresAuth: true}

	task1, _ := q.AddTask(cmd, check)
	task2, _ := q.AddTask(cmd, check)

	// Approve one
	q.ApproveTask(task1.ID)

	// Get pending
	pending := q.GetPendingTasks()
	if len(pending) != 1 {
		t.Errorf("Expected 1 pending task, got %d", len(pending))
	}

	if pending[0].ID != task2.ID {
		t.Error("Wrong task is pending")
	}
}

func TestQueue_GetTasksBySession(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tada-queue-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	queueFile := filepath.Join(tmpDir, "queue.json")
	q := NewQueue(queueFile, "session-123")

	cmd := ai.Command{Cmd: "rm", Args: []string{"-rf", "/tmp/test"}}
	check := &security.CheckResult{Allowed: true, RequiresAuth: true}

	q.AddTask(cmd, check)

	tasks := q.GetTasksBySession("session-123")
	if len(tasks) != 1 {
		t.Errorf("Expected 1 task for session-123, got %d", len(tasks))
	}

	tasks = q.GetTasksBySession("other-session")
	if len(tasks) != 0 {
		t.Errorf("Expected 0 tasks for other session, got %d", len(tasks))
	}
}

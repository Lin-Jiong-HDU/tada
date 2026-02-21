package queue

import (
	"testing"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
	"github.com/Lin-Jiong-HDU/tada/internal/core/security"
)

func TestTaskStatus_String(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected string
	}{
		{TaskStatusPending, "pending"},
		{TaskStatusApproved, "approved"},
		{TaskStatusRejected, "rejected"},
		{TaskStatusExecuting, "executing"},
		{TaskStatusCompleted, "completed"},
		{TaskStatusFailed, "failed"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, tt.status)
		}
	}
}

func TestNewTask(t *testing.T) {
	cmd := ai.Command{Cmd: "rm", Args: []string{"-rf", "/tmp/test"}}
	checkResult := &security.CheckResult{
		Allowed:      true,
		RequiresAuth: true,
		Warning:      "Dangerous command",
		Reason:       "In dangerous list",
	}

	task := NewTask("session-123", cmd, checkResult)

	if task.SessionID != "session-123" {
		t.Error("Expected SessionID to be session-123")
	}
	if task.Status != TaskStatusPending {
		t.Error("Expected initial status to be pending")
	}
	if task.ID == "" {
		t.Error("Expected ID to be generated")
	}
	if task.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
}

func TestTask_CanTransitionTo(t *testing.T) {
	tests := []struct {
		from     TaskStatus
		to       TaskStatus
		expected bool
	}{
		{TaskStatusPending, TaskStatusApproved, true},
		{TaskStatusPending, TaskStatusRejected, true},
		{TaskStatusApproved, TaskStatusExecuting, true},
		{TaskStatusExecuting, TaskStatusCompleted, true},
		{TaskStatusExecuting, TaskStatusFailed, true},
		{TaskStatusPending, TaskStatusCompleted, false},
		{TaskStatusRejected, TaskStatusApproved, false},
		{TaskStatusCompleted, TaskStatusPending, false},
	}

	for _, tt := range tests {
		task := &Task{Status: tt.from}
		result := task.CanTransitionTo(tt.to)
		if result != tt.expected {
			t.Errorf("Transition %s -> %s: expected %v, got %v",
				tt.from, tt.to, tt.expected, result)
		}
	}
}

func TestExecutionResult(t *testing.T) {
	result := ExecutionResult{
		ExitCode: 0,
		Output:   "success",
	}

	if result.ExitCode != 0 {
		t.Error("Expected ExitCode 0")
	}
	if result.Output != "success" {
		t.Error("Expected output 'success'")
	}
}

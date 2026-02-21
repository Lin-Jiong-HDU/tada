package queue

import (
	"time"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
	"github.com/Lin-Jiong-HDU/tada/internal/core/security"
	"github.com/google/uuid"
)

// TaskStatus represents the current state of a task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"   // Waiting for authorization
	TaskStatusApproved  TaskStatus = "approved"  // Authorized by user
	TaskStatusRejected  TaskStatus = "rejected"  // Rejected by user
	TaskStatusExecuting TaskStatus = "executing" // Currently executing
	TaskStatusCompleted TaskStatus = "completed" // Execution completed successfully
	TaskStatusFailed    TaskStatus = "failed"    // Execution failed
)

// Task represents a command awaiting or executed authorization
type Task struct {
	ID          string                `json:"id"`
	SessionID   string                `json:"session_id"`
	Command     ai.Command            `json:"command"`
	CheckResult *security.CheckResult `json:"check_result"`
	Status      TaskStatus            `json:"status"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
	Result      *ExecutionResult      `json:"result,omitempty"`
}

// ExecutionResult holds the result of a command execution
type ExecutionResult struct {
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output"`
	Error    string `json:"error,omitempty"`
}

// NewTask creates a new task with pending status
func NewTask(sessionID string, cmd ai.Command, checkResult *security.CheckResult) *Task {
	now := time.Now()
	return &Task{
		ID:          uuid.New().String(),
		SessionID:   sessionID,
		Command:     cmd,
		CheckResult: checkResult,
		Status:      TaskStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// CanTransitionTo checks if a status transition is valid
func (t *Task) CanTransitionTo(newStatus TaskStatus) bool {
	validTransitions := map[TaskStatus][]TaskStatus{
		TaskStatusPending:   {TaskStatusApproved, TaskStatusRejected},
		TaskStatusApproved:  {TaskStatusExecuting},
		TaskStatusExecuting: {TaskStatusCompleted, TaskStatusFailed},
	}

	allowed, exists := validTransitions[t.Status]
	if !exists {
		return false
	}

	for _, status := range allowed {
		if status == newStatus {
			return true
		}
	}
	return false
}

// TransitionStatus updates the task status if the transition is valid
func (t *Task) TransitionStatus(newStatus TaskStatus) bool {
	if !t.CanTransitionTo(newStatus) {
		return false
	}
	t.Status = newStatus
	t.UpdatedAt = time.Now()
	return true
}

// SetResult records the execution result
func (t *Task) SetResult(result *ExecutionResult) {
	t.Result = result
	t.UpdatedAt = time.Now()
}

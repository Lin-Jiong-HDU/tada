package queue

import (
	"fmt"
	"sync"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
	"github.com/Lin-Jiong-HDU/tada/internal/core/security"
)

// Manager manages the task queue with persistence
type Manager struct {
	sessionID string
	store     *Store
	tasks     []*Task
	mu        sync.RWMutex
}

// NewQueue creates a new queue manager
func NewQueue(filePath string, sessionID string) *Manager {
	store := NewStore(filePath)

	// Load existing tasks
	tasks, _ := store.Load()

	return &Manager{
		sessionID: sessionID,
		store:     store,
		tasks:     tasks,
	}
}

// AddTask adds a new task to the queue
func (m *Manager) AddTask(cmd ai.Command, checkResult *security.CheckResult) (*Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task := NewTask(m.sessionID, cmd, checkResult)
	m.tasks = append(m.tasks, task)

	if err := m.store.Save(m.tasks); err != nil {
		return nil, err
	}

	return task, nil
}

// GetAllTasks returns all tasks
func (m *Manager) GetAllTasks() []*Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make([]*Task, len(m.tasks))
	copy(result, m.tasks)
	return result
}

// GetPendingTasks returns all pending tasks
func (m *Manager) GetPendingTasks() []*Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var pending []*Task
	for _, task := range m.tasks {
		if task.Status == TaskStatusPending {
			pending = append(pending, task)
		}
	}
	return pending
}

// GetTasksBySession returns tasks for a specific session
func (m *Manager) GetTasksBySession(sessionID string) []*Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Task
	for _, task := range m.tasks {
		if task.SessionID == sessionID {
			result = append(result, task)
		}
	}
	return result
}

// ApproveTask approves a task for execution
func (m *Manager) ApproveTask(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, task := range m.tasks {
		if task.ID == taskID {
			if !task.TransitionStatus(TaskStatusApproved) {
				return fmt.Errorf("cannot transition task %s from %s to approved",
					taskID, task.Status)
			}
			return m.store.Save(m.tasks)
		}
	}
	return fmt.Errorf("task not found: %s", taskID)
}

// RejectTask rejects a task
func (m *Manager) RejectTask(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, task := range m.tasks {
		if task.ID == taskID {
			if !task.TransitionStatus(TaskStatusRejected) {
				return fmt.Errorf("cannot transition task %s from %s to rejected",
					taskID, task.Status)
			}
			return m.store.Save(m.tasks)
		}
	}
	return fmt.Errorf("task not found: %s", taskID)
}

// MarkExecuting marks a task as executing
func (m *Manager) MarkExecuting(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, task := range m.tasks {
		if task.ID == taskID {
			task.TransitionStatus(TaskStatusExecuting)
			return m.store.Save(m.tasks)
		}
	}
	return fmt.Errorf("task not found: %s", taskID)
}

// SetTaskResult records the execution result for a task
func (m *Manager) SetTaskResult(taskID string, result *ExecutionResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, task := range m.tasks {
		if task.ID == taskID {
			task.SetResult(result)

			// Update status based on result - validate transition
			var targetStatus TaskStatus
			if result.ExitCode == 0 && result.Error == "" {
				targetStatus = TaskStatusCompleted
			} else {
				targetStatus = TaskStatusFailed
			}

			if !task.CanTransitionTo(targetStatus) {
				return fmt.Errorf("cannot transition task %s from %s to %s",
					taskID, task.Status, targetStatus)
			}

			task.TransitionStatus(targetStatus)
			return m.store.Save(m.tasks)
		}
	}
	return fmt.Errorf("task not found: %s", taskID)
}

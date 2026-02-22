package execution

import (
	"context"
	"fmt"

	"github.com/Lin-Jiong-HDU/tada/internal/core"
	"github.com/Lin-Jiong-HDU/tada/internal/core/queue"
)

// TaskExecutor executes queued tasks
type TaskExecutor struct {
	queue    *queue.Manager
	executor *core.Executor
}

// NewTaskExecutor creates a new task executor
func NewTaskExecutor(q *queue.Manager, executor *core.Executor) *TaskExecutor {
	return &TaskExecutor{
		queue:    q,
		executor: executor,
	}
}

// ExecuteTask executes a single task by ID
func (e *TaskExecutor) ExecuteTask(ctx context.Context, taskID string) error {
	// Get the task
	tasks := e.queue.GetAllTasks()
	var target *queue.Task
	for _, task := range tasks {
		if task.ID == taskID {
			target = task
			break
		}
	}

	if target == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// Check if can transition to executing
	if !target.CanTransitionTo(queue.TaskStatusExecuting) {
		return fmt.Errorf("task %s cannot be executed (current status: %s)",
			taskID, target.Status)
	}

	// Mark as executing
	if err := e.queue.MarkExecuting(taskID); err != nil {
		return fmt.Errorf("failed to mark executing: %w", err)
	}

	// Execute the command
	result, err := e.executor.Execute(ctx, target.Command)

	// Convert result to queue result
	queueResult := &queue.ExecutionResult{
		ExitCode: result.ExitCode,
		Output:   result.Output,
	}

	// Check for errors from the result (Execute always returns nil for err)
	if result.Error != nil {
		queueResult.Error = result.Error.Error()
	}

	// If Execute returned an error (shouldn't happen but handle defensively), incorporate it
	if err != nil {
		if queueResult.Error == "" {
			queueResult.Error = err.Error()
		} else {
			queueResult.Error = fmt.Sprintf("%s: %v", queueResult.Error, err)
		}
	}

	// Ensure a non-zero exit code when an error is present
	if queueResult.Error != "" && queueResult.ExitCode == 0 {
		queueResult.ExitCode = 1
	}

	// Set result (will transition to completed/failed)
	if err := e.queue.SetTaskResult(taskID, queueResult); err != nil {
		return fmt.Errorf("failed to set result: %w", err)
	}

	return nil
}

// ExecuteAllApproved executes all approved tasks
func (e *TaskExecutor) ExecuteAllApproved(ctx context.Context) ([]*queue.ExecutionResult, error) {
	tasks := e.queue.GetAllTasks()

	var results []*queue.ExecutionResult
	var lastErr error

	for _, task := range tasks {
		if task.Status == queue.TaskStatusApproved {
			if err := e.ExecuteTask(ctx, task.ID); err != nil {
				lastErr = err
				// Continue with other tasks even if one fails
			} else {
				// Get updated task with result
				updatedTasks := e.queue.GetAllTasks()
				for _, ut := range updatedTasks {
					if ut.ID == task.ID && ut.Result != nil {
						results = append(results, ut.Result)
						break
					}
				}
			}
		}
	}

	return results, lastErr
}

// ExecuteResult returns the execution results
func (e *TaskExecutor) ExecuteResult() []*queue.ExecutionResult {
	tasks := e.queue.GetAllTasks()
	var results []*queue.ExecutionResult

	for _, task := range tasks {
		if task.Result != nil {
			results = append(results, task.Result)
		}
	}

	return results
}

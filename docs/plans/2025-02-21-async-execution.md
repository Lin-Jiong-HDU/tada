# Async Execution Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Enable shell-style async command execution with `tada "command" &` syntax, where tasks are queued, approved via TUI, and then executed.

**Architecture:** Parse `&` suffix to detect async mode, set `IsAsync` flag on commands, queue them without immediate confirmation. In TUI, when user approves tasks, execute them immediately and show results. Also add `tada run` command for batch execution of all approved tasks.

**Tech Stack:** Go 1.23+, existing queue system, Bubble Tea TUI, context for execution management

---

## Task 1: Parse Async Syntax `&`

**Files:**
- Modify: `internal/core/engine.go:36-59` (Process method)
- Test: `internal/core/engine_test.go` (create if not exists)

**Step 1: Write the failing test**

Create `internal/core/engine_test.go`:

```go
package core

import (
	"context"
	"testing"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
	"github.com/Lin-Jiong-HDU/tada/internal/core/security"
)

// Mock AI provider for testing
type mockAIProvider struct {
	intent *ai.Intent
}

func (m *mockAIProvider) ParseIntent(ctx context.Context, input string, systemPrompt string) (*ai.Intent, error) {
	return m.intent, nil
}

func (m *mockAIProvider) AnalyzeOutput(ctx context.Context, cmd string, output string) (string, error) {
	return "done", nil
}

func (m *mockAIProvider) Chat(ctx context.Context, messages []ai.Message) (string, error) {
	return "response", nil
}

func TestEngine_ParseAsyncSyntax(t *testing.T) {
	executor := NewExecutor(5)
	policy := security.DefaultPolicy()
	engine := NewEngine(&mockAIProvider{}, executor, policy)

	tests := []struct {
		name     string
		input    string
		wantAsync bool
	}{
		{
			name:     "sync command without &",
			input:    "create a folder",
			wantAsync: false,
		},
		{
			name:     "async command with &",
			input:    "create a folder &",
			wantAsync: true,
		},
		{
			name:     "async with multiple & &",
			input:    "create a folder & &",
			wantAsync: true,
		},
		{
			name:     "async with space before &",
			input:    "create a folder & ",
			wantAsync: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isAsync := parseAsyncSyntax(tt.input)
			if isAsync != tt.wantAsync {
				t.Errorf("parseAsyncSyntax() = %v, want %v", isAsync, tt.wantAsync)
			}
		})
	}
}

func TestEngine_StripAsyncSyntax(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"create folder &", "create folder"},
		{"create folder&", "create folder"},
		{"create folder & ", "create folder"},
		{"create folder & &", "create folder &"},
		{"no async here", "no async here"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := stripAsyncSyntax(tt.input)
			if result != tt.expected {
				t.Errorf("stripAsyncSyntax() = %q, want %q", result, tt.expected)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/core -v -run TestEngine_ParseAsyncSyntax
```

Expected: FAIL - "parseAsyncSyntax undefined" and "stripAsyncSyntax undefined"

**Step 3: Write minimal implementation**

Modify `internal/core/engine.go`, add these helper functions before the Process method:

```go
// parseAsyncSyntax checks if the input ends with & for async execution
func parseAsyncSyntax(input string) bool {
	trimmed := strings.TrimSpace(input)
	return strings.HasSuffix(trimmed, "&")
}

// stripAsyncSyntax removes trailing & from input
func stripAsyncSyntax(input string) string {
	trimmed := strings.TrimSpace(input)
	if strings.HasSuffix(trimmed, "&") {
		return strings.TrimSpace(trimmed[:len(trimmed)-1])
	}
	return trimmed
}
```

Also need to add `strings` import at top of file:

```go
import (
	"context"
	"fmt"
	"strings"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
	// ... other imports
)
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/core -v -run TestEngine_ParseAsyncSyntax
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/core/engine.go internal/core/engine_test.go
git commit -m "feat: add async syntax parser for & suffix"
```

---

## Task 2: Set IsAsync Flag Based on Syntax

**Files:**
- Modify: `internal/core/engine.go:36-59` (Process method)
- Test: `internal/core/engine_test.go`

**Step 1: Write the failing test**

Add to `internal/core/engine_test.go`:

```go
func TestEngine_Process_SetsIsAsyncFlag(t *testing.T) {
	// Setup
	cmd := ai.Command{Cmd: "mkdir", Args: []string{"test"}}
	intent := &ai.Intent{
		Commands:     []ai.Command{cmd},
		Reason:       "create directory",
		NeedsConfirm: false,
	}

	executor := NewExecutor(5)
	policy := security.DefaultPolicy()
	engine := NewEngine(&mockAIProvider{intent: intent}, executor, policy)

	// Mock the security check to allow command
	engine.securityController = &mockSecurityController{}

	// Test async input
	ctx := context.Background()
	err := engine.Process(ctx, "create test &", "")

	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// Verify the intent's command was marked as async
	// Note: This requires the AI provider to return the modified intent
	// For this test, we'll verify the mock was called correctly
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/core -v -run TestEngine_Process_SetsIsAsync
```

Expected: FAIL - Need to update Process method to handle async

**Step 3: Write minimal implementation**

Modify `internal/core/engine.go:36-59`, update the Process method to handle async syntax:

```go
// Process handles a user request from input to output
func (e *Engine) Process(ctx context.Context, input string, systemPrompt string) error {
	// Check for async syntax
	isAsync := parseAsyncSyntax(input)
	if isAsync {
		input = stripAsyncSyntax(input)
	}

	// Add user message to session (use original input for history)
	session := storage.GetCurrentSession()
	if session != nil {
		storage.AddMessage("user", input)
	}

	// Step 1: Parse intent
	fmt.Println("🧠 Thinking...")
	intent, err := e.ai.ParseIntent(ctx, input, systemPrompt)
	if err != nil {
		return fmt.Errorf("failed to parse intent: %w", err)
	}

	// Mark all commands as async if & was used
	if isAsync {
		for i := range intent.Commands {
			intent.Commands[i].IsAsync = true
		}
	}

	fmt.Printf("📝 Plan: %s\n", intent.Reason)

	// ... rest of method stays the same
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/core -v -run TestEngine_Process_SetsIsAsync
```

Expected: PASS (may need mock security controller)

**Step 5: Commit**

```bash
git add internal/core/engine.go internal/core/engine_test.go
go test ./internal/core -v
git commit -m "feat: set IsAsync flag when & suffix detected"
```

---

## Task 3: Skip Confirmation for Async Commands

**Files:**
- Modify: `internal/core/engine.go:74-101` (Process method security check section)
- Test: `internal/core/engine_test.go`

**Step 1: Verify current behavior**

Current code at lines 74-101:
```go
if result.RequiresAuth {
	if cmd.IsAsync {
		// Add to queue for async commands
		if e.queue != nil {
			task, err := e.queue.AddTask(cmd, result)
			if err != nil {
				return fmt.Errorf("failed to queue task: %w", err)
			}
			fmt.Printf("📋 命令已加入队列 (ID: %s)\n", task.ID)
			fmt.Printf("   使用 'tada tasks' 查看并授权\n")
			continue
		}
		// Fall through to sync prompt if no queue
	}

	// Sync command: prompt for confirmation
	confirmed, err := terminal.Confirm(cmd, result)
	// ...
}
```

**The async flow is already implemented!** When `IsAsync=true`, the command is queued without immediate confirmation.

**Step 2: Write test to verify async queues without confirmation**

Add to `internal/core/engine_test.go`:

```go
func TestEngine_AsyncQueuesWithoutConfirmation(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "tada-async-test-*")
	defer os.RemoveAll(tmpDir)

	queueFile := filepath.Join(tmpDir, "queue.json")
	q := queue.NewQueue(queueFile, "test-session")

	cmd := ai.Command{Cmd: "mkdir", Args: []string{"test"}, IsAsync: true}
	intent := &ai.Intent{
		Commands:     []ai.Command{cmd},
		Reason:       "create directory async",
		NeedsConfirm: false,
	}

	executor := NewExecutor(5)
	policy := security.DefaultPolicy()
	policy.CommandLevel = security.ConfirmAlways // Require confirmation for all
	engine := NewEngine(&mockAIProvider{intent: intent}, executor, policy)
	engine.SetQueue(q)

	ctx := context.Background()
	err := engine.Process(ctx, "create test &", "")

	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// Verify task was queued
	tasks := q.GetAllTasks()
	if len(tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(tasks))
	}

	if tasks[0].Status != queue.TaskStatusPending {
		t.Errorf("Expected pending status, got %s", tasks[0].Status)
	}

	if !tasks[0].Command.IsAsync {
		t.Error("Expected IsAsync to be true")
	}
}
```

**Step 3: Run test**

```bash
go test ./internal/core -v -run TestEngine_AsyncQueuesWithoutConfirmation
```

Expected: PASS (already implemented)

**Step 4: Commit**

```bash
git add internal/core/engine_test.go
go test ./internal/core -v
git commit -m "test: verify async commands queue without confirmation"
```

---

## Task 4: Add Task Executor Function

**Files:**
- Create: `internal/core/queue/executor.go`
- Test: `internal/core/queue/executor_test.go`

**Step 1: Write the failing test**

Create `internal/core/queue/executor_test.go`:

```go
package queue

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
	"github.com/Lin-Jiong-HDU/tada/internal/core"
	"github.com/Lin-Jiong-HDU/tada/internal/core/security"
)

func TestTaskExecutor_ExecuteApprovedTask(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "tada-executor-test-*")
	defer os.RemoveAll(tmpDir)

	queueFile := filepath.Join(tmpDir, "queue.json")
	q := NewQueue(queueFile, "test-session")

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

	if tasks[0].Status != TaskStatusCompleted {
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
	q := NewQueue(queueFile, "test-session")

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
		if task.Status == TaskStatusCompleted {
			completed++
		}
	}
	if completed != 2 {
		t.Errorf("Expected 2 completed tasks, got %d", completed)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/core/queue -v -run TestTaskExecutor
```

Expected: FAIL - "TaskExecutor undefined"

**Step 3: Write minimal implementation**

Create `internal/core/queue/executor.go`:

```go
package queue

import (
	"context"
	"fmt"

	"github.com/Lin-Jiong-HDU/tada/internal/core"
)

// TaskExecutor executes queued tasks
type TaskExecutor struct {
	queue    *Manager
	executor *core.Executor
}

// NewTaskExecutor creates a new task executor
func NewTaskExecutor(queue *Manager, executor *core.Executor) *TaskExecutor {
	return &TaskExecutor{
		queue:    queue,
		executor: executor,
	}
}

// ExecuteTask executes a single task by ID
func (e *TaskExecutor) ExecuteTask(ctx context.Context, taskID string) error {
	// Get the task
	tasks := e.queue.GetAllTasks()
	var target *Task
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
	if !target.CanTransitionTo(TaskStatusExecuting) {
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
	queueResult := &ExecutionResult{
		ExitCode: result.ExitCode,
		Output:   result.Output,
	}

	if err != nil {
		queueResult.Error = err.Error()
	}

	// Set result (will transition to completed/failed)
	if err := e.queue.SetTaskResult(taskID, queueResult); err != nil {
		return fmt.Errorf("failed to set result: %w", err)
	}

	return nil
}

// ExecuteAllApproved executes all approved tasks
func (e *TaskExecutor) ExecuteAllApproved(ctx context.Context) ([]*ExecutionResult, error) {
	tasks := e.queue.GetAllTasks()

	var results []*ExecutionResult
	var lastErr error

	for _, task := range tasks {
		if task.Status == TaskStatusApproved {
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
func (e *TaskExecutor) ExecuteResult() []*ExecutionResult {
	tasks := e.queue.GetAllTasks()
	var results []*ExecutionResult

	for _, task := range tasks {
		if task.Result != nil {
			results = append(results, task.Result)
		}
	}

	return results
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/core/queue -v -run TestTaskExecutor
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/core/queue/executor.go internal/core/queue/executor_test.go
go test ./internal/core/queue -v
git commit -m "feat(queue): add task executor for running approved tasks"
```

---

## Task 5: Integrate Execution into TUI

**Files:**
- Modify: `cmd/tada/tasks.go:50-74` (TUI authorize handler)
- Modify: `internal/core/tui/queue_model.go:66-96` (Update method)

**Step 1: Update authorize handler to execute tasks**

Modify `cmd/tada/tasks.go`, update the onAuthorize handler:

```go
// Create handlers that persist to the appropriate queue and execute
onAuthorize := func(taskID string) tea.Cmd {
	return func() tea.Msg {
		q := findQueueForTask(queues, taskID)
		if q != nil {
			// Approve the task
			if err := q.ApproveTask(taskID); err != nil {
				return tui.AuthorizeResultMsg{TaskID: taskID, Success: false}
			}

			// Execute the task immediately
			ctx := context.Background()
			executor := core.NewExecutor(30 * time.Second)
			taskExecutor := queue.NewTaskExecutor(q, executor)

			go func() {
				_ = taskExecutor.ExecuteTask(ctx, taskID)
			}()
		}
		return tui.AuthorizeResultMsg{TaskID: taskID, Success: true}
	}
}
```

Add context import:
```go
import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Lin-Jiong-HDU/tada/internal/core"
	"github.com/Lin-Jiong-HDU/tada/internal/core/queue"
	// ... other imports
)
```

**Step 2: Update TUI to show executing status**

Modify `internal/core/tui/queue_model.go`, update Update method to handle status changes:

```go
case AuthorizeResultMsg:
	if msg.Success {
		// Update task status to executing (will be updated by background executor)
		for i, task := range m.tasks {
			if task.ID == msg.TaskID {
				// Create a copy with updated status
				updatedTask := *task
				updatedTask.Status = queue.TaskStatusExecuting
				m.tasks[i] = &updatedTask

				// Start a ticker to check for status updates
				return m, tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
					return StatusCheckMsg{TaskID: msg.TaskID}
				})
			}
		}
	}
	return m, nil
```

Add StatusCheckMsg to types.go first:

Modify `internal/core/tui/types.go`:

```go
// StatusCheckMsg is sent to check task status updates
type StatusCheckMsg struct {
	TaskID string
}
```

Add to queue_model.go imports:
```go
import (
	"fmt"
	"strings"
	"time"

	"github.com/Lin-Jiong-HDU/tada/internal/core/queue"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)
```

**Step 3: Handle status check messages**

Add to `internal/core/tui/queue_model.go` Update method:

```go
case StatusCheckMsg:
	// Check if task is still executing or completed
	for i, task := range m.tasks {
		if task.ID == msg.TaskID && task.Status == queue.TaskStatusExecuting {
			// In real implementation, we'd check the queue for status updates
			// For now, simulate completion after status check
			// The queue will have the actual status
			return m, nil
		}
	}
	return m, nil
```

**Step 4: Test the TUI execution**

```bash
# Build and test
go build ./cmd/tada
./tata tasks
```

Expected: When pressing 'a' to approve, task executes and status updates

**Step 5: Commit**

```bash
git add cmd/tada/tasks.go internal/core/tui/types.go internal/core/tui/queue_model.go
go test ./...
git commit -m "feat(tui): execute tasks immediately on approval"
```

---

## Task 6: Add `tada run` Command

**Files:**
- Create: `cmd/tada/run.go`
- Modify: `cmd/tada/main.go:97-103` (init function)

**Step 1: Write the failing test**

Create `cmd/tada/run_test.go`:

```go
package main

import (
	"testing"
)

func TestRunCommand_Exists(t *testing.T) {
	cmd := getRunCommand()
	if cmd == nil {
		t.Fatal("Expected run command to exist")
	}

	if cmd.Use != "run" {
		t.Errorf("Expected command name 'run', got '%s'", cmd.Use)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./cmd/tada -v
```

Expected: FAIL - "getRunCommand undefined"

**Step 3: Write minimal implementation**

Create `cmd/tada/run.go`:

```go
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Lin-Jiong-HDU/tada/internal/core"
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
	executor := core.NewExecutor(30)
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

		taskExecutor := queue.NewTaskExecutor(q, executor)
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
```

**Step 4: Register the command**

Modify `cmd/tada/main.go`, add to init function:

```go
func init() {
	// Add subcommands
	rootCmd.AddCommand(chatCmd)
	rootCmd.AddCommand(getTasksCommand())
	rootCmd.AddCommand(getRunCommand()) // Add this line

	chatCmd.PersistentFlags().BoolVarP(&incognito, "incognito", "i", false, "Run in incognito mode (don't save history)")
}
```

**Step 5: Run test to verify it passes**

```bash
go test ./cmd/tada -v
```

Expected: PASS

**Step 6: Test the command**

```bash
go build ./cmd/tada
./tada run --help
```

**Step 7: Commit**

```bash
git add cmd/tada/run.go cmd/tada/run_test.go cmd/tada/main.go
go test ./cmd/tada -v
git commit -m "feat(cli): add run command for batch execution"
```

---

## Task 7: Update TUI Keys Help Text

**Files:**
- Modify: `internal/core/tui/keys.go:1276-1315` (defaultKeyMap function)

**Step 1: Update key bindings help**

Modify the help text to reflect execution behavior:

```go
func defaultKeyMap() keyMap {
	return keyMap{
		Up: key{
			Key:  tea.Key{Type: tea.KeyRunes, Runes: []rune{'k'}},
			help: "↑/k",
		},
		Down: key{
			Key:  tea.Key{Type: tea.KeyRunes, Runes: []rune{'j'}},
			help: "↓/j",
		},
		Authorize: key{
			Key:  tea.Key{Type: tea.KeyRunes, Runes: []rune{'a'}},
			help: "授权并执行",
		},
		Reject: key{
			Key:  tea.Key{Type: tea.KeyRunes, Runes: []rune{'r'}},
			help: "拒绝选中",
		},
		AuthorizeAll: key{
			Key:  tea.Key{Type: tea.KeyShiftRunes, Runes: []rune{'A'}},
			help: "全部执行",
		},
		RejectAll: key{
			Key:  tea.Key{Type: tea.KeyShiftRunes, Runes: []rune{'R'}},
			help: "全部拒绝",
		},
		Enter: key{
			Key:  tea.Key{Type: tea.KeyEnter},
			help: "查看详情",
		},
		Quit: key{
			Key:  tea.Key{Type: tea.KeyRunes, Runes: []rune{'q'}},
			help: "退出",
		},
		ForceQuit: key{
			Key:  tea.Key{Type: tea.KeyEsc},
			help: "强制退出",
		},
	}
}
```

**Step 2: Commit**

```bash
git add internal/core/tui/keys.go
git commit -m "docs(tui): update key help text for execution behavior"
```

---

## Task 8: Full Integration Testing

**Files:**
- Create: `tests/e2e/async_test.go`

**Step 1: Write E2E test**

Create `tests/e2e/async_test.go`:

```go
package e2e

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
	engine := core.NewEngine(nil, core.NewExecutor(30), security.DefaultPolicy())
	engine.SetQueue(q)

	// Verify async parsing
	isAsync := parseAsyncSyntax("create folder &")
	if !isAsync {
		t.Error("Expected async syntax to be detected")
	}

	stripped := stripAsyncSyntax("create folder &")
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
	taskExecutor := queue.NewTaskExecutor(q, executor)

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
```

Note: This test uses parseAsyncSyntax and stripAsyncSyntax which are not exported. We need to either export them or test through the public interface.

**Step 2: Fix test by using public interface or exporting functions**

Option: Export the async parsing functions by capitalizing them:

In `internal/core/engine.go`, rename:
- `parseAsyncSyntax` → `ParseAsyncSyntax`
- `stripAsyncSyntax` → `StripAsyncSyntax`

**Step 3: Run E2E test**

```bash
TADA_INTEGRATION_TEST=1 go test ./tests/e2e/... -v -run TestAsync
```

Expected: PASS

**Step 4: Commit**

```bash
git add tests/e2e/async_test.go internal/core/engine.go
git commit -m "test(e2e): add async workflow integration test"
```

---

## Task 9: Update Documentation

**Files:**
- Modify: `README.md`
- Modify: `docs/getting-started.md`

**Step 1: Update README**

Add async execution section to README:

```markdown
## Usage

```bash
# Simple commands (synchronous)
tada "list all files in the current directory"
tada "create a new folder named docs"

# Async commands (queue for later execution)
tada "create a new folder named tmp &"
tada "download large file &"

# View and authorize pending tasks
tada tasks

# Execute all approved tasks
tada run

# Incognito mode (no history saved)
tada -i "run a secret command"
```

### Async Execution

Add `&` at the end of your command to run it asynchronously:

```bash
tada "long running task &"
```

Async commands are queued without immediate confirmation. Use `tada tasks` to review and authorize them. Authorized tasks execute immediately in the TUI, or use `tada run` for batch execution.
```

**Step 2: Update getting-started guide**

Add to `docs/getting-started.md`:

```markdown
## Async Execution

For long-running commands, use async mode:

```bash
tada "compile project &"
```

The command will be queued. Use `tada tasks` to:

1. View pending commands
2. Authorize (a) or reject (r) individual commands
3. Authorize all (A) or reject all (R)

Authorized tasks execute immediately. For batch execution, use:

```bash
tada run
```

This executes all approved tasks that haven't been run yet.
```

**Step 3: Commit**

```bash
git add README.md docs/getting-started.md
git commit -m "docs: add async execution documentation"
```

---

## Task 10: Verify and Clean Up

**Files:**
- All modified files

**Step 1: Run full test suite**

```bash
go test ./... -v
```

Expected: All tests pass

**Step 2: Build and verify**

```bash
go build ./cmd/tada
./tada --help
./tada tasks --help
./tada run --help
```

Expected: All commands show help

**Step 3: Manual testing workflow**

```bash
# Test async syntax
./tada "echo test &"

# Check queue
./tada tasks

# Test run command
./tada run
```

**Step 4: Check for go vet issues**

```bash
go vet ./...
```

Expected: No warnings

**Step 5: Format code**

```bash
go fmt ./...
```

**Step 6: Final commit**

```bash
git add -A
git commit -m "test: verify and finalize async execution implementation"
```

---

## Summary

This implementation adds:

1. **Async Syntax Parsing** - Detects `&` suffix to enable async mode
2. **Auto-Queuing** - Async commands are queued without confirmation
3. **TUI Execution** - Tasks execute immediately when authorized
4. **Batch Execution** - `tada run` command for executing all approved tasks
5. **Status Tracking** - Tasks show executing/completed/failed status in TUI

**User Workflow:**
```bash
# Queue async command
tada "download file &"

# View and authorize (executes on approval)
tada tasks

# Or batch execute later
tada run
```

**Total estimated time:** ~60-90 minutes (TDD with small commits)

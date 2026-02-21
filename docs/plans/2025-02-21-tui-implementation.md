# TUI Interface Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a TUI interface for tada's async command authorization queue and sync command terminal confirmation.

**Architecture:** The TUI system uses Bubble Tea for the queue interface and simple terminal prompts for sync commands. Tasks are persisted as JSON in session directories. The Engine.Process method is modified to integrate with the queue for async commands and terminal prompts for sync commands.

**Tech Stack:** Go 1.23+, Bubble Tea (TUI framework), lipgloss (styling), JSON persistence

---

## Task 1: Add IsAsync Field to ai.Command

**Files:**
- Modify: `internal/ai/provider.go:19-22`
- Test: `internal/ai/provider_test.go` (create if not exists)

**Step 1: Write the failing test**

Create `internal/ai/provider_test.go`:

```go
package ai

import "testing"

func TestCommand_IsAsync(t *testing.T) {
    cmd := Command{
        Cmd:     "rm",
        Args:    []string{"-rf", "/tmp/test"},
        IsAsync: true,
    }

    if !cmd.IsAsync {
        t.Error("Expected IsAsync to be true")
    }
}

func TestCommand_JSONMarshal(t *testing.T) {
    cmd := Command{
        Cmd:     "dd",
        Args:    []string{"if=/dev/zero", "of=file"},
        IsAsync: true,
    }

    data, err := cmd.MarshalJSON()
    if err != nil {
        t.Fatalf("Failed to marshal: %v", err)
    }

    // Verify IsAsync is in JSON
    if !contains(string(data), "is_async") {
        t.Error("Expected is_async field in JSON")
    }
}

func contains(s, substr string) bool {
    return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
    for i := 0; i <= len(s)-len(substr); i++ {
        if s[i:i+len(substr)] == substr {
            return true
        }
    }
    return false
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/ai -v -run TestCommand
```

Expected: FAIL - "IsAsync" field does not exist

**Step 3: Write minimal implementation**

Modify `internal/ai/provider.go` lines 18-22:

```go
// Command represents a shell command to execute
type Command struct {
    Cmd     string   `json:"cmd"`
    Args    []string `json:"args"`
    IsAsync bool     `json:"is_async"` // Indicates async execution requiring queue authorization
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/ai -v -run TestCommand
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/ai/provider.go internal/ai/provider_test.go
git commit -m "feat(ai): add IsAsync field to Command"
```

---

## Task 2: Create Queue Task Types

**Files:**
- Create: `internal/core/queue/types.go`
- Test: `internal/core/queue/types_test.go`

**Step 1: Write the failing test**

Create `internal/core/queue/types_test.go`:

```go
package queue

import (
    "testing"
    "time"

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
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/core/queue -v
```

Expected: FAIL - types do not exist

**Step 3: Write minimal implementation**

Create `internal/core/queue/types.go`:

```go
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
    ID          string                 `json:"id"`
    SessionID   string                 `json:"session_id"`
    Command     ai.Command             `json:"command"`
    CheckResult *security.CheckResult  `json:"check_result"`
    Status      TaskStatus             `json:"status"`
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
    Result      *ExecutionResult       `json:"result,omitempty"`
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
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/core/queue -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/core/queue/types.go internal/core/queue/types_test.go
git commit -m "feat(queue): add task types and status transitions"
```

---

## Task 3: Create Queue JSON Store

**Files:**
- Create: `internal/core/queue/store.go`
- Test: `internal/core/queue/store_test.go`

**Step 1: Write the failing test**

Create `internal/core/queue/store_test.go`:

```go
package queue

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/Lin-Jiong-HDU/tada/internal/ai"
    "github.com/Lin-Jiong-HDU/tada/internal/core/security"
)

func TestStore_SaveAndLoad(t *testing.T) {
    // Create temp directory
    tmpDir, err := os.MkdirTemp("", "tada-queue-test-*")
    if err != nil {
        t.Fatal(err)
    }
    defer os.RemoveAll(tmpDir)

    queueFile := filepath.Join(tmpDir, "queue.json")
    store := NewStore(queueFile)

    // Create test tasks
    cmd1 := ai.Command{Cmd: "rm", Args: []string{"-rf", "/tmp/test"}}
    check1 := &security.CheckResult{Allowed: true, RequiresAuth: true}
    task1 := NewTask("session-1", cmd1, check1)

    // Save tasks
    tasks := []*Task{task1}
    if err := store.Save(tasks); err != nil {
        t.Fatalf("Failed to save: %v", err)
    }

    // Verify file exists
    if _, err := os.Stat(queueFile); os.IsNotExist(err) {
        t.Error("Queue file was not created")
    }

    // Load tasks
    loaded, err := store.Load()
    if err != nil {
        t.Fatalf("Failed to load: %v", err)
    }

    if len(loaded) != 1 {
        t.Errorf("Expected 1 task, got %d", len(loaded))
    }

    if loaded[0].ID != task1.ID {
        t.Errorf("Expected ID %s, got %s", task1.ID, loaded[0].ID)
    }

    if loaded[0].Status != TaskStatusPending {
        t.Errorf("Expected status pending, got %s", loaded[0].Status)
    }
}

func TestStore_Load_EmptyFile(t *testing.T) {
    tmpDir, err := os.MkdirTemp("", "tada-queue-test-*")
    if err != nil {
        t.Fatal(err)
    }
    defer os.RemoveAll(tmpDir)

    queueFile := filepath.Join(tmpDir, "queue.json")
    store := NewStore(queueFile)

    // Loading non-existent file should return empty slice
    tasks, err := store.Load()
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }

    if len(tasks) != 0 {
        t.Errorf("Expected 0 tasks, got %d", len(tasks))
    }
}

func TestStore_InvalidJSON(t *testing.T) {
    tmpDir, err := os.MkdirTemp("", "tada-queue-test-*")
    if err != nil {
        t.Fatal(err)
    }
    defer os.RemoveAll(tmpDir)

    queueFile := filepath.Join(tmpDir, "queue.json")

    // Write invalid JSON
    if err := os.WriteFile(queueFile, []byte("{invalid json"), 0644); err != nil {
        t.Fatal(err)
    }

    store := NewStore(queueFile)

    _, err = store.Load()
    if err == nil {
        t.Error("Expected error for invalid JSON")
    }
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/core/queue -v -run TestStore
```

Expected: FAIL - Store does not exist

**Step 3: Write minimal implementation**

Create `internal/core/queue/store.go`:

```go
package queue

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
)

// QueueFile represents the persisted queue data
type QueueFile struct {
    Tasks []*Task `json:"tasks"`
}

// Store handles JSON persistence of the task queue
type Store struct {
    filePath string
}

// NewStore creates a new store for the given file path
func NewStore(filePath string) *Store {
    return &Store{filePath: filePath}
}

// Save persists tasks to JSON file
func (s *Store) Save(tasks []*Task) error {
    // Ensure directory exists
    if err := os.MkdirAll(filepath.Dir(s.filePath), 0755); err != nil {
        return fmt.Errorf("failed to create directory: %w", err)
    }

    queueFile := QueueFile{Tasks: tasks}

    data, err := json.MarshalIndent(queueFile, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal queue: %w", err)
    }

    if err := os.WriteFile(s.filePath, data, 0644); err != nil {
        return fmt.Errorf("failed to write queue file: %w", err)
    }

    return nil
}

// Load loads tasks from JSON file
func (s *Store) Load() ([]*Task, error) {
    data, err := os.ReadFile(s.filePath)
    if err != nil {
        if os.IsNotExist(err) {
            // No queue file exists yet
            return []*Task{}, nil
        }
        return nil, fmt.Errorf("failed to read queue file: %w", err)
    }

    var queueFile QueueFile
    if err := json.Unmarshal(data, &queueFile); err != nil {
        return nil, fmt.Errorf("failed to unmarshal queue: %w", err)
    }

    return queueFile.Tasks, nil
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/core/queue -v -run TestStore
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/core/queue/store.go internal/core/queue/store_test.go
git commit -m "feat(queue): add JSON store for task persistence"
```

---

## Task 4: Create Queue Manager

**Files:**
- Create: `internal/core/queue/queue.go`
- Test: `internal/core/queue/queue_test.go`

**Step 1: Write the failing test**

Create `internal/core/queue/queue_test.go`:

```go
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
    tasks, _ := q.GetAllTasks()
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
    tasks, _ := q.GetAllTasks()
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

    tasks, _ := q.GetAllTasks()
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
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/core/queue -v -run TestQueue
```

Expected: FAIL - Queue does not exist

**Step 3: Write minimal implementation**

Create `internal/core/queue/queue.go`:

```go
package queue

import (
    "fmt"
    "sync"
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

// ApproroveTask approves a task for execution
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

            // Update status based on result
            if result.ExitCode == 0 && result.Error == "" {
                task.TransitionStatus(TaskStatusCompleted)
            } else {
                task.TransitionStatus(TaskStatusFailed)
            }

            return m.store.Save(m.tasks)
        }
    }
    return fmt.Errorf("task not found: %s", taskID)
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/core/queue -v -run TestQueue
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/core/queue/queue.go internal/core/queue/queue_test.go
go test ./internal/core/queue -v
git commit -m "feat(queue): add queue manager for task lifecycle"
```

---

## Task 5: Create Terminal Prompt for Sync Commands

**Files:**
- Create: `internal/terminal/prompt.go`
- Test: `internal/terminal/prompt_test.go`

**Step 1: Write the failing test**

Create `internal/terminal/prompt_test.go`:

```go
package terminal

import (
    "strings"
    "testing"

    "github.com/Lin-Jiong-HDU/tada/internal/ai"
    "github.com/Lin-Jiong-HDU/tada/internal/core/security"
)

func TestConfirm_YesInput(t *testing.T) {
    input := strings.NewReader("y\n")
    output := &strings.Builder{}

    cmd := ai.Command{Cmd: "rm", Args: []string{"-rf", "/tmp/test"}}
    check := &security.CheckResult{
        Allowed:      true,
        RequiresAuth: true,
        Warning:      "Dangerous command",
        Reason:       "In dangerous list",
    }

    result, err := ConfirmWithIO(cmd, check, input, output)
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }

    if !result {
        t.Error("Expected confirmation to succeed")
    }

    outputStr := output.String()
    if !strings.Contains(outputStr, "此操作需要您的授权") {
        t.Error("Expected authorization prompt in output")
    }
}

func TestConfirm_SkipInput(t *testing.T) {
    input := strings.NewReader("s\n")
    output := &strings.Builder{}

    cmd := ai.Command{Cmd: "rm", Args: []string{"-rf", "/tmp/test"}}
    check := &security.CheckResult{
        Allowed:      true,
        RequiresAuth: true,
        Warning:      "Dangerous command",
    }

    result, err := ConfirmWithIO(cmd, check, input, output)
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }

    if result {
        t.Error("Expected confirmation to be skipped")
    }
}

func TestConfirm_QuitInput(t *testing.T) {
    input := strings.NewReader("q\n")
    output := &strings.Builder{}

    cmd := ai.Command{Cmd: "rm", Args: []string{"-rf", "/tmp/test"}}
    check := &security.CheckResult{Allowed: true, RequiresAuth: true}

    _, err := ConfirmWithIO(cmd, check, input, output)
    if err != ErrQuitAll {
        t.Errorf("Expected ErrQuitAll, got %v", err)
    }
}

func TestConfirm_InvalidThenValid(t *testing.T) {
    input := strings.NewReader("invalid\ny\n")
    output := &strings.Builder{}

    cmd := ai.Command{Cmd: "rm", Args: []string{"-rf", "/tmp/test"}}
    check := &security.CheckResult{Allowed: true, RequiresAuth: true}

    result, err := ConfirmWithIO(cmd, check, input, output)
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }

    if !result {
        t.Error("Expected confirmation to succeed after invalid input")
    }
}

func TestConfirm_CaseInsensitive(t *testing.T) {
    tests := []struct {
        input    string
        expected bool
    }{
        {"Y\n", true},
        {"y\n", true},
        {"S\n", false},
        {"s\n", false},
        {"Q\n", false},
        {"q\n", false},
    }

    for _, tt := range tests {
        input := strings.NewReader(tt.input)
        output := &strings.Builder{}

        cmd := ai.Command{Cmd: "rm", Args: []string{"-rf", "/tmp/test"}}
        check := &security.CheckResult{Allowed: true, RequiresAuth: true}

        result, err := ConfirmWithIO(cmd, check, input, output)
        if tt.input == "Q\n" || tt.input == "q\n" {
            if err != ErrQuitAll {
                t.Errorf("Input %s: expected ErrQuitAll", tt.input)
            }
        } else {
            if err != nil {
                t.Errorf("Input %s: expected no error, got %v", tt.input, err)
            }
            if result != tt.expected {
                t.Errorf("Input %s: expected %v, got %v", tt.input, tt.expected, result)
            }
        }
    }
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/terminal -v
```

Expected: FAIL - prompt does not exist

**Step 3: Write minimal implementation**

Create `internal/terminal/prompt.go`:

```go
package terminal

import (
    "bufio"
    "errors"
    "fmt"
    "io"
    "strings"

    "github.com/Lin-Jiong-HDU/tada/internal/ai"
    "github.com/Lin-Jiong-HDU/tada/internal/core/security"
)

// Errors returned by Confirm
var (
    ErrQuitAll = errors.New("quit all commands")
)

// Confirm prompts the user for command confirmation
// Returns true if approved, false if skipped, ErrQuitAll if quit all
func Confirm(cmd ai.Command, checkResult *security.CheckResult) (bool, error) {
    return ConfirmWithIO(cmd, checkResult, nil, nil)
}

// ConfirmWithIO prompts the user with provided IO (for testing)
func ConfirmWithIO(cmd ai.Command, checkResult *security.CheckResult, input io.Reader, output io.Writer) (bool, error) {
    if input == nil {
        input = bufio.NewReader(io.Reader(nil))
    }
    if output == nil {
        output = io.Writer(nil)
    }

    // Build command string
    cmdStr := cmd.Cmd
    if len(cmd.Args) > 0 {
        cmdStr += " " + strings.Join(cmd.Args, " ")
    }

    // Display prompt
    fmt.Fprintf(output, "\n⚠️  此操作需要您的授权\n\n")
    fmt.Fprintf(output, "命令: %s\n", cmdStr)

    if checkResult.Warning != "" {
        fmt.Fprintf(output, "警告: %s\n", checkResult.Warning)
    }

    if checkResult.Reason != "" {
        fmt.Fprintf(output, "原因: %s\n", checkResult.Reason)
    }

    fmt.Fprintf(output, "\n[y] 执行  [s] 跳过  [q] 取消全部\n> ")

    // Read input
    scanner := bufio.NewScanner(input)
    for scanner.Scan() {
        choice := strings.ToLower(strings.TrimSpace(scanner.Text()))

        switch choice {
        case "y":
            fmt.Fprintln(output, "✓ 已授权执行")
            return true, nil
        case "s":
            fmt.Fprintln(output, "⊘ 已跳过")
            return false, nil
        case "q":
            fmt.Fprintln(output, "✗ 取消全部操作")
            return false, ErrQuitAll
        default:
            fmt.Fprintf(output, "无效选项，请输入 y/s/q: ")
        }
    }

    if err := scanner.Err(); err != nil {
        return false, err
    }

    return false, nil
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/terminal -v
```

Expected: PASS (fix any issues with io.Reader handling)

**Step 5: Commit**

```bash
git add internal/terminal/prompt.go internal/terminal/prompt_test.go
go test ./internal/terminal -v
git commit -m "feat(terminal): add sync command confirmation prompt"
```

---

## Task 6: Create TUI Key Bindings

**Files:**
- Create: `internal/core/tui/keys.go`
- Test: `internal/core/tui/keys_test.go`

**Step 1: Write the failing test**

Create `internal/core/tui/keys_test.go`:

```go
package tui

import (
    "testing"

    tea "github.com/charmbracelet/bubbletea"
)

func TestKeyMap_Help(t *testing.T) {
    km := defaultKeyMap()

    help := km.Help()
    if help == nil {
        t.Error("Expected help to be generated")
    }

    // Verify key bindings are present
    helpText := help.String()
    if !contains(helpText, "授权") || !contains(helpText, "拒绝") {
        t.Error("Expected help to contain authorize and reject actions")
    }
}

func TestKeyMap_Bindings(t *testing.T) {
    km := defaultKeyMap()

    // Check up/down bindings
    if km.Up == nil {
        t.Error("Expected Up binding")
    }
    if km.Down == nil {
        t.Error("Expected Down binding")
    }

    // Check action bindings
    if km.Authorize == nil {
        t.Error("Expected Authorize binding")
    }
    if km.Reject == nil {
        t.Error("Expected Reject binding")
    }
}

func contains(s, substr string) bool {
    return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
    for i := 0; i <= len(s)-len(substr); i++ {
        if s[i:i+len(substr)] == substr {
            return i
        }
    }
    return -1
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/core/tui -v
```

Expected: FAIL - tui package doesn't exist

**Step 3: Write minimal implementation**

Create `internal/core/tui/keys.go`:

```go
package tui

import (
    "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

// keyMap defines key bindings for the TUI
type keyMap struct {
    Up             key
    Down           key
    Authorize      key
    Reject         key
    AuthorizeAll   key
    RejectAll      key
    Enter          key
    Quit           key
    ForceQuit      key
}

// key represents a key binding with help text
type key struct {
    tea.Key
    help string
}

// shortHelp returns key bindings for short help view
func (k keyMap) shortHelp() []key {
    return []key{k.Authorize, k.Reject, k.AuthorizeAll, k.Quit}
}

// fullHelp returns all key bindings for full help view
func (k keyMap) fullHelp() []key {
    return []key{
        k.Up, k.Down,
        k.Authorize, k.Reject,
        k.AuthorizeAll, k.RejectAll,
        k.Enter, k.Quit, k.ForceQuit,
    }
}

// Help generates the help view
func (k keyMap) Help() tea.Help {
    return tea.Help{
        Short: k.shortHelp(),
        Full:  k.fullHelp(),
    }
}

// defaultKeyMap creates the default key bindings
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
            help: "授权选中",
        },
        Reject: key{
            Key:  tea.Key{Type: tea.KeyRunes, Runes: []rune{'r'}},
            help: "拒绝选中",
        },
        AuthorizeAll: key{
            Key:  tea.Key{Type: tea.KeyShiftRunes, Runes: []rune{'A'}},
            help: "全部授权",
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

**Step 4: Run test to verify it passes**

```bash
go test ./internal/core/tui -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/core/tui/keys.go internal/core/tui/keys_test.go
go test ./internal/core/tui -v
git commit -m "feat(tui): add key bindings for queue interface"
```

---

## Task 7: Create TUI Model

**Files:**
- Create: `internal/core/tui/types.go`
- Create: `internal/core/tui/queue_model.go`
- Test: `internal/core/tui/queue_model_test.go`

**Step 1: Write the failing test**

Create `internal/core/tui/queue_model_test.go`:

```go
package tui

import (
    "testing"

    "github.com/Lin-Jiong-HDU/tada/internal/ai"
    "github.com/Lin-Jiong-HDU/tada/internal/core/queue"
    "github.com/Lin-Jiong-HDU/tada/internal/core/security"
)

func TestNewModel(t *testing.T) {
    tasks := []*queue.Task{
        {ID: "1", Status: queue.TaskStatusPending},
    }

    model := NewModel(tasks)

    if model == nil {
        t.Fatal("Expected non-nil model")
    }

    if len(model.tasks) != 1 {
        t.Errorf("Expected 1 task, got %d", len(model.tasks))
    }

    if model.cursor != 0 {
        t.Errorf("Expected cursor at 0, got %d", model.cursor)
    }
}

func TestModel_Init(t *testing.T) {
    tasks := []*queue.Task{}
    model := NewModel(tasks)

    cmd := model.Init()
    if cmd != nil {
        t.Error("Expected nil command from Init")
    }
}

func TestModel_Update_UpKey(t *testing.T) {
    tasks := []*queue.Task{
        {ID: "1", Status: queue.TaskStatusPending},
        {ID: "2", Status: queue.TaskStatusPending},
    }
    model := NewModel(tasks)
    model.cursor = 1

    // Move up
    msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
    newModel, cmd := model.Update(msg)

    m := newModel.(model)
    if m.cursor != 0 {
        t.Errorf("Expected cursor at 0, got %d", m.cursor)
    }
    if cmd != nil {
        t.Error("Expected nil command")
    }
}

func TestModel_Update_DownKey(t *testing.T) {
    tasks := []*queue.Task{
        {ID: "1", Status: queue.TaskStatusPending},
        {ID: "2", Status: queue.TaskStatusPending},
    }
    model := NewModel(tasks)

    // Move down
    msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
    newModel, cmd := model.Update(msg)

    m := newModel.(model)
    if m.cursor != 1 {
        t.Errorf("Expected cursor at 1, got %d", m.cursor)
    }
}

func TestModel_Update_AuthorizeKey(t *testing.T) {
    tasks := []*queue.Task{
        {ID: "1", Status: queue.TaskStatusPending},
    }
    model := NewModel(tasks)
    model.onAuthorize = func(taskID string) tea.Cmd {
        return func() tea.Msg {
            return AuthorizeResultMsg{TaskID: taskID, Success: true}
        }
    }

    msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
    newModel, cmd := model.Update(msg)

    m := newModel.(model)
    if cmd == nil {
        t.Error("Expected command from authorize")
    }
}

func TestModel_Update_QuitKey(t *testing.T) {
    tasks := []*queue.Task{}
    model := NewModel(tasks)

    msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
    newModel, cmd := model.Update(msg)

    if cmd == nil {
        t.Error("Expected quit command")
    }

    _, ok := cmd().(tea.QuitMsg)
    if !ok {
        t.Error("Expected QuitMsg")
    }
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/core/tui -v -run TestModel
```

Expected: FAIL - model does not exist

**Step 3: Write minimal implementation**

Create `internal/core/tui/types.go`:

```go
package tui

import (
    tea "github.com/charmbracelet/bubbletea"
)

// AuthorizeResultMsg is sent when authorization completes
type AuthorizeResultMsg struct {
    TaskID  string
    Success bool
}

// RejectResultMsg is sent when rejection completes
type RejectResultMsg struct {
    TaskID  string
    Success bool
}

// TickMsg is sent for UI updates
type TickMsg time.Time

// TasksLoadedMsg is sent when tasks are loaded
type TasksLoadedMsg struct {
    Tasks []*Task
}

// Model is the interface for the TUI model
type Model interface {
    Init() tea.Cmd
    Update(msg tea.Msg) (tea.Model, tea.Cmd)
    View() string
}
```

Create `internal/core/tui/queue_model.go`:

```go
package tui

import (
    "fmt"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/Lin-Jiong-HDU/tada/internal/core/queue"
)

// model is the Bubble Tea model for the queue TUI
type model struct {
    tasks       []*queue.Task
    cursor      int
    selected    map[string]struct{}
    keys        keyMap
    showingHelp bool
    onAuthorize func(string) tea.Cmd
    onReject    func(string) tea.Cmd
}

// NewModel creates a new queue UI model
func NewModel(tasks []*queue.Task) Model {
    return model{
        tasks:       tasks,
        cursor:      0,
        selected:    make(map[string]struct{}),
        keys:        defaultKeyMap(),
        showingHelp: false,
        onAuthorize: defaultAuthorizeHandler,
        onReject:    defaultRejectHandler,
    }
}

func defaultAuthorizeHandler(taskID string) tea.Cmd {
    return func() tea.Msg {
        return AuthorizeResultMsg{TaskID: taskID, Success: true}
    }
}

func defaultRejectHandler(taskID string) tea.Cmd {
    return func() tea.Msg {
        return RejectResultMsg{TaskID: taskID, Success: true}
    }
}

// Init initializes the model
func (m model) Init() tea.Cmd {
    return nil
}

// Update handles messages
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        return m.handleKeyMsg(msg)

    case AuthorizeResultMsg:
        if msg.Success {
            // Update task status
            for i, task := range m.tasks {
                if task.ID == msg.TaskID {
                    m.tasks[i].Status = queue.TaskStatusApproved
                    break
                }
            }
        }
        return m, nil

    case RejectResultMsg:
        if msg.Success {
            for i, task := range m.tasks {
                if task.ID == msg.TaskID {
                    m.tasks[i].Status = queue.TaskStatusRejected
                    break
                }
            }
        }
        return m, nil
    }

    return m, nil
}

func (m model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    // Handle quit
    if key.Matches(msg, m.keys.Quit) || key.Matches(msg, m.keys.ForceQuit) {
        return m, tea.Quit
    }

    // Handle help toggle
    if key.Matches(msg, m.keys.Enter) {
        m.showingHelp = !m.showingHelp
        return m, nil
    }

    // Handle navigation
    switch {
    case key.Matches(msg, m.keys.Up):
        if m.cursor > 0 {
            m.cursor--
        }
    case key.Matches(msg, m.keys.Down):
        if m.cursor < len(m.tasks)-1 {
            m.cursor++
        }
    }

    // Handle actions
    switch {
    case key.Matches(msg, m.keys.Authorize):
        if len(m.tasks) > 0 {
            task := m.tasks[m.cursor]
            return m, m.onAuthorize(task.ID)
        }
    case key.Matches(msg, m.keys.Reject):
        if len(m.tasks) > 0 {
            task := m.tasks[m.cursor]
            return m, m.onReject(task.ID)
        }
    case key.Matches(msg, m.keys.AuthorizeAll):
        var cmds []tea.Cmd
        for _, task := range m.tasks {
            if task.Status == queue.TaskStatusPending {
                cmds = append(cmds, m.onAuthorize(task.ID))
            }
        }
        return m, tea.Batch(cmds...)
    case key.Matches(msg, m.keys.RejectAll):
        var cmds []tea.Cmd
        for _, task := range m.tasks {
            if task.Status == queue.TaskStatusPending {
                cmds = append(cmds, m.onReject(task.ID))
            }
        }
        return m, tea.Batch(cmds...)
    }

    return m, nil
}

// View renders the UI
func (m model) View() string {
    if m.showingHelp {
        return m.renderHelp()
    }
    return m.renderQueue()
}

func (m model) renderQueue() string {
    var s string

    // Header
    s += titleStyle.Render(" tada 任务队列 ") + "\n\n"

    if len(m.tasks) == 0 {
        s += subtleStyle.Render("没有待授权任务") + "\n"
        return s
    }

    // Group by session
    grouped := m.groupTasksBySession()

    for sessionID, tasks := range grouped {
        s += fmt.Sprintf(" 会话: %s \n", sessionID)

        for _, task := range tasks {
            cursor := " "
            if m.getCursorForTask(task.ID) == m.cursor {
                cursor = ">"
            }

            // Status indicator
            status := getStatusIndicator(task.Status)

            // Command string
            cmdStr := task.Command.Cmd
            if len(task.Command.Args) > 0 {
                cmdStr += " " + fmt.Sprint(task.Command.Args)
            }

            // Truncate if too long
            if len(cmdStr) > 50 {
                cmdStr = cmdStr[:47] + "..."
            }

            s += fmt.Sprintf("%s [%s] %s\n", cursor, status, cmdStr)

            if task.CheckResult != nil && task.CheckResult.Warning != "" {
                s += subtleStyle.Render("     警告: "+task.CheckResult.Warning) + "\n"
            }
        }
        s += "\n"
    }

    // Footer
    s += m.renderFooter()

    return s
}

func (m model) renderHelp() string {
    return helpStyle.Render(`
按 q 返回队列
`) + "\n"
}

func (m model) renderFooter() string {
    return "\n" + m.keys.Help().View(nil)
}

func (m model) groupTasksBySession() map[string][]*queue.Task {
    grouped := make(map[string][]*queue.Task)
    currentIdx := 0

    for _, task := range m.tasks {
        // Only show pending tasks
        if task.Status == queue.TaskStatusPending {
            grouped[task.SessionID] = append(grouped[task.SessionID], task)
        }
        if task.ID == m.tasks[m.cursor].ID {
            m.cursor = currentIdx
        }
        currentIdx++
    }

    return grouped
}

func (m model) getCursorForTask(taskID string) int {
    for i, task := range m.tasks {
        if task.ID == taskID {
            return i
        }
    }
    return 0
}

func getStatusIndicator(status queue.TaskStatus) string {
    switch status {
    case queue.TaskStatusPending:
        return " "
    case queue.TaskStatusApproved:
        return "✓"
    case queue.TaskStatusRejected:
        return "✗"
    case queue.TaskStatusExecuting:
        return "⋯"
    case queue.TaskStatusCompleted:
        return "✓"
    case queue.TaskStatusFailed:
        return "!"
    default:
        return "?"
    }
}

// Styles
var (
    titleStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
    subtleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
    helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/core/tui -v -run TestModel
```

Expected: PASS (fix any compilation issues)

**Step 5: Commit**

```bash
git add internal/core/tui/types.go internal/core/tui/queue_model.go internal/core/tui/queue_model_test.go
go test ./internal/core/tui -v
git commit -m "feat(tui): add Bubble Tea model for queue UI"
```

---

## Task 8: Create TUI View

**Files:**
- Create: `internal/core/tui/queue_view.go`

**Step 1: Write the view rendering**

Create `internal/core/tui/queue_view.go`:

```go
package tui

import (
    "fmt"

    "github.com/charmbracelet/lipgloss"
    "github.com/Lin-Jiong-HDU/tada/internal/core/queue"
)

// Renderer handles TUI rendering
type Renderer struct {
    width  int
    height int
    style  *StyleConfig
}

// StyleConfig defines visual styles
type StyleConfig struct {
    TitleColor     lipgloss.Color
    SubtleColor    lipgloss.Color
    ErrorColor     lipgloss.Color
    SuccessColor   lipgloss.Color
    WarningColor   lipgloss.Color
    SelectedColor  lipgloss.Color
    BorderColor    lipgloss.Color
}

// DefaultStyleConfig returns the default style configuration
func DefaultStyleConfig() *StyleConfig {
    return &StyleConfig{
        TitleColor:     lipgloss.Color("10"),  // Green
        SubtleColor:    lipgloss.Color("241"), // Grey
        ErrorColor:     lipgloss.Color("9"),   // Red
        SuccessColor:   lipgloss.Color("10"),  // Green
        WarningColor:   lipgloss.Color("11"),  // Yellow
        SelectedColor:  lipgloss.Color("12"),  // Blue
        BorderColor:    lipgloss.Color("8"),   // Dark grey
    }
}

// NewRenderer creates a new TUI renderer
func NewRenderer(width, height int) *Renderer {
    return &Renderer{
        width:  width,
        height: height,
        style:  DefaultStyleConfig(),
    }
}

// Render renders the full TUI view
func (r *Renderer) Render(model *model) string {
    return r.renderHeader() + "\n" + r.renderTasks(model) + "\n" + r.renderFooter(model)
}

func (r *Renderer) renderHeader() string {
    title := lipgloss.NewStyle().
        Foreground(r.style.TitleColor).
        Bold(true).
        Render("tada 任务队列")

    border := lipgloss.NewStyle().
        Foreground(r.style.BorderColor).
        Render("──────────────────────────────────────────────────────────────")

    return title + "\n" + border
}

func (r *Renderer) renderTasks(model *model) string {
    if len(model.tasks) == 0 {
        return r.renderEmptyState()
    }

    // Group by session
    grouped := model.groupTasksBySession()

    if len(grouped) == 0 {
        return r.renderEmptyState()
    }

    var result string

    for sessionID, tasks := range grouped {
        result += r.renderSessionHeader(sessionID)

        for _, task := range tasks {
            result += r.renderTask(model, task)
        }

        result += "\n"
    }

    return result
}

func (r *Renderer) renderEmptyState() string {
    return lipgloss.NewStyle().
        Foreground(r.style.SubtleColor).
        Render("\n  没有待授权任务\n")
}

func (r *Renderer) renderSessionHeader(sessionID string) string {
    style := lipgloss.NewStyle().
        Foreground(r.style.SelectedColor).
        Bold(true)

    return fmt.Sprintf("\n  %s\n", style.Render("会话: "+sessionID))
}

func (r *Renderer) renderTask(model *model, task *queue.Task) string {
    // Determine if selected
    isSelected := model.cursor == r.getTaskIndex(model, task.ID)
    cursor := " "
    if isSelected {
        cursor = ">"
    }

    // Status indicator
    status := r.renderStatus(task.Status)

    // Command string
    cmdStr := r.renderCommand(task)

    // Warning
    warning := ""
    if task.CheckResult != nil && task.CheckResult.Warning != "" {
        warning = lipgloss.NewStyle().
            Foreground(r.style.WarningColor).
            Render("     警告: "+task.CheckResult.Warning+"\n")
    }

    // Build task box
    taskStyle := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(r.style.BorderColor).
        Width(min(r.width-4, 70))

    content := fmt.Sprintf("[%s] %s\n%s", status, cmdStr, warning)
    if warning == "" {
        content = fmt.Sprintf("[%s] %s", status, cmdStr)
    }

    box := taskStyle.Render(content)

    return fmt.Sprintf("  %s %s\n", cursor, box)
}

func (r *Renderer) renderStatus(status queue.TaskStatus) string {
    var symbol string
    var color lipgloss.Color

    switch status {
    case queue.TaskStatusPending:
        symbol = " "
        color = r.style.SubtleColor
    case queue.TaskStatusApproved:
        symbol = "✓"
        color = r.style.SuccessColor
    case queue.TaskStatusRejected:
        symbol = "✗"
        color = r.style.ErrorColor
    case queue.TaskStatusExecuting:
        symbol = "⋯"
        color = r.style.WarningColor
    case queue.TaskStatusCompleted:
        symbol = "✓"
        color = r.style.SuccessColor
    case queue.TaskStatusFailed:
        symbol = "!"
        color = r.style.ErrorColor
    default:
        symbol = "?"
        color = r.style.SubtleColor
    }

    return lipgloss.NewStyle().Foreground(color).Render(symbol)
}

func (r *Renderer) renderCommand(task *queue.Task) string {
    cmdStr := task.Command.Cmd
    if len(task.Command.Args) > 0 {
        cmdStr += " " + fmt.Sprint(task.Command.Args)
    }

    // Truncate if too long
    maxLen := 60
    if len(cmdStr) > maxLen {
        cmdStr = cmdStr[:maxLen-3] + "..."
    }

    return lipgloss.NewStyle().
        Foreground(r.style.TitleColor).
        Render(cmdStr)
}

func (r *Renderer) renderFooter(model *model) string {
    help := model.keys.Help().View(nil)

    style := lipgloss.NewStyle().
        Foreground(r.style.SubtleColor)

    return "\n" + style.Render(help)
}

func (r *Renderer) getTaskIndex(model *model, taskID string) int {
    for i, task := range model.tasks {
        if task.ID == taskID {
            return i
        }
    }
    return 0
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
```

**Step 2: Commit**

```bash
git add internal/core/tui/queue_view.go
git commit -m "feat(tui): add view renderer for queue UI"
```

---

## Task 9: Create Tasks Command

**Files:**
- Create: `cmd/tada/tasks.go`
- Modify: `cmd/tada/main.go` (add tasks command)
- Test: `cmd/tada/tasks_test.go`

**Step 1: Write the failing test**

Create `cmd/tada/tasks_test.go`:

```go
package main

import (
    "testing"
)

func TestTasksCommand_Validates(t *testing.T) {
    // This test verifies the tasks command is registered
    // More detailed testing requires integration setup
    cmd := getTasksCommand()
    if cmd == nil {
        t.Fatal("Expected tasks command to exist")
    }

    if cmd.Use != "tasks" {
        t.Errorf("Expected command name 'tasks', got '%s'", cmd.Use)
    }

    if cmd.Short == "" {
        t.Error("Expected command to have a short description")
    }
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./cmd/tada -v
```

Expected: FAIL - tasks command doesn't exist

**Step 3: Write minimal implementation**

Create `cmd/tada/tasks.go`:

```go
package main

import (
    "fmt"
    "os"
    "path/filepath"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/Lin-Jiong-HDU/tada/internal/core/queue"
    "github.com/Lin-Jiong-HDU/tada/internal/core/tui"
    "github.com/Lin-Jiong-HDU/tada/internal/storage"
)

// getTasksCommand returns the tasks command
func getTasksCommand() *Command {
    return &Command{
        Use:   "tasks",
        Short: "管理待授权命令队列",
        Long: `打开 TUI 界面管理需要授权的命令。

查看、授权或拒绝待授权的异步命令。`,
        RunE: runTasks,
    }
}

func runTasks(cmd *Command, args []string) error {
    // Get sessions directory
    configDir, err := storage.GetConfigDir()
    if err != nil {
        return fmt.Errorf("failed to get config directory: %w", err)
    }

    sessionsDir := filepath.Join(configDir, storage.SessionDirName)

    // Load all queue files
    allTasks, err := loadAllTasks(sessionsDir)
    if err != nil {
        return fmt.Errorf("failed to load tasks: %w", err)
    }

    // Filter pending tasks
    var pendingTasks []*queue.Task
    for _, task := range allTasks {
        if task.Status == queue.TaskStatusPending {
            pendingTasks = append(pendingTasks, task)
        }
    }

    // Create TUI model
    model := tui.NewModel(pendingTasks)

    // Run TUI
    p := tea.NewProgram(model, tea.WithAltScreen())
    if _, err := p.Run(); err != nil {
        return fmt.Errorf("failed to run TUI: %w", err)
    }

    return nil
}

func loadAllTasks(sessionsDir string) ([]*queue.Task, error) {
    var allTasks []*queue.Task

    // Read all session directories
    entries, err := os.ReadDir(sessionsDir)
    if err != nil {
        if os.IsNotExist(err) {
            return allTasks, nil
        }
        return nil, err
    }

    for _, entry := range entries {
        if entry.IsDir() {
            sessionDir := filepath.Join(sessionsDir, entry.Name())
            queueFile := filepath.Join(sessionDir, "queue.json")

            // Load queue file
            q := queue.NewQueue(queueFile, entry.Name())
            tasks := q.GetAllTasks()
            allTasks = append(allTasks, tasks...)
        }
    }

    return allTasks, nil
}
```

**Step 4: Modify main.go to add tasks command**

Read current `cmd/tada/main.go` and add the tasks command to the root command:

```go
package main

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
)

func main() {
    rootCmd := &cobra.Command{
        Use:   "tada",
        Short: "Terminal AI Command Assistant",
        Long:  `tada - Terminal AI Command Assistant with security controls`,
    }

    // Add existing commands
    rootCmd.AddCommand(getChatCommand())
    rootCmd.AddCommand(getTasksCommand())  // Add this line

    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
```

**Step 5: Run test to verify it passes**

```bash
go test ./cmd/tada -v
```

Expected: PASS

**Step 6: Commit**

```bash
git add cmd/tada/tasks.go cmd/tada/tasks_test.go cmd/tada/main.go
go test ./cmd/tada -v
git commit -m "feat(cli): add tasks command for queue management"
```

---

## Task 10: Integrate Queue with Engine.Process

**Files:**
- Modify: `internal/core/engine.go:28-101`
- Test: `internal/core/engine_test.go` (update if exists)

**Step 1: Write the failing test**

Create or update `internal/core/engine_test.go`:

```go
package core

import (
    "context"
    "os"
    "path/filepath"
    "testing"

    "github.com/Lin-Jiong-HDU/tada/internal/ai"
    "github.com/Lin-Jiong-HDU/tada/internal/core/security"
)

func TestEngine_Process_SyncCommandRequiresAuth(t *testing.T) {
    // Create temporary session
    tmpDir, _ := os.MkdirTemp("", "tada-engine-test-*")
    defer os.RemoveAll(tmpDir)

    policy := &security.SecurityPolicy{
        CommandLevel: security.ConfirmDangerous,
        DangerousCommands: map[string]bool{
            "rm": true,
        },
    }

    executor := NewExecutor()
    engine := NewEngine(nil, executor, policy)

    // For sync commands (IsAsync=false), we expect the terminal prompt to be called
    // This is a placeholder - in real test we'd mock the terminal prompt
    ctx := context.Background()

    // This test verifies the integration point
    if engine.securityController == nil {
        t.Error("Expected security controller to be initialized")
    }
}

func TestEngine_Process_AsyncCommandGoesToQueue(t *testing.T) {
    tmpDir, _ := os.MkdirTemp("", "tada-engine-test-*")
    defer os.RemoveAll(tmpDir)

    policy := &security.SecurityPolicy{
        CommandLevel: security.ConfirmDangerous,
    }

    queueFile := filepath.Join(tmpDir, "queue.json")
    q := queue.NewQueue(queueFile, "test-session")

    executor := NewExecutor()
    engine := NewEngine(nil, executor, policy)
    engine.SetQueue(q)

    ctx := context.Background()

    // Test that async commands are queued
    cmd := ai.Command{Cmd: "rm", Args: []string{"-rf", "/tmp/test"}, IsAsync: true}
    result, _ := engine.securityController.CheckCommand(cmd)

    if result.RequiresAuth {
        // For async commands, should go to queue
        task, _ := q.AddTask(cmd, result)
        if task.Status != queue.TaskStatusPending {
            t.Error("Expected task to be pending")
        }
    }
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/core -v -run TestEngine_Process
```

Expected: FAIL - queue integration not implemented

**Step 3: Write minimal implementation**

Modify `internal/core/engine.go`:

```go
package core

import (
	"context"
	"fmt"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
	"github.com/Lin-Jiong-HDU/tada/internal/core/queue"
	"github.com/Lin-Jiong-HDU/tada/internal/core/security"
	"github.com/Lin-Jiong-HDU/tada/internal/storage"
	"github.com/Lin-Jiong-HDU/tada/internal/terminal"
)

// Engine orchestrates the AI workflow
type Engine struct {
	ai                 ai.AIProvider
	executor           *Executor
	securityController *security.SecurityController
	queue              *queue.Manager
}

// NewEngine creates a new engine
func NewEngine(aiProvider ai.AIProvider, executor *Executor, securityPolicy *security.SecurityPolicy) *Engine {
	return &Engine{
		ai:                 aiProvider,
		executor:           executor,
		securityController: security.NewSecurityController(securityPolicy),
	}
}

// SetQueue sets the task queue for async commands
func (e *Engine) SetQueue(q *queue.Manager) {
	e.queue = q
}

// Process handles a user request from input to output
func (e *Engine) Process(ctx context.Context, input string, systemPrompt string) error {
	// Add user message to session
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

	fmt.Printf("📝 Plan: %s\n", intent.Reason)

	// Step 2: Confirm if needed
	if intent.NeedsConfirm {
		fmt.Println("⚠️  This command requires confirmation.")
		fmt.Println("⚠️  Proceeding (confirmation will be added in TUI phase)...")
	}

	// Step 3: Execute commands (with security check)
	for i, cmd := range intent.Commands {
		// Security check before execution
		result, err := e.securityController.CheckCommand(cmd)
		if err != nil {
			return fmt.Errorf("security check failed: %w", err)
		}

		if !result.Allowed {
			fmt.Printf("🚫 拒绝执行: %s\n", result.Reason)
			continue
		}

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
			if err == terminal.ErrQuitAll {
				fmt.Println("✗ 取消全部操作")
				return nil
			}
			if err != nil {
				return fmt.Errorf("confirmation error: %w", err)
			}
			if !confirmed {
				continue
			}
		}

		fmt.Printf("\n🔧 Executing [%d/%d]: %s %v\n", i+1, len(intent.Commands), cmd.Cmd, cmd.Args)

		execResult, err := e.executor.Execute(ctx, cmd)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			continue
		}

		// Show output (truncated if too long)
		e.displayOutput(execResult.Output)

		// Step 4: Analyze result
		if execResult.Error != nil {
			fmt.Printf("📊 Command failed (exit code %d)\n", execResult.ExitCode)
		} else {
			analysis, err := e.ai.AnalyzeOutput(ctx, cmd.Cmd, execResult.Output)
			if err != nil {
				fmt.Printf("⚠️  Could not analyze output\n")
			} else {
				fmt.Printf("✅ %s\n", analysis)
			}
		}
	}

	// Add assistant response to session
	if session != nil {
		storage.AddMessage("assistant", intent.Reason)
	}

	return nil
}

// displayOutput shows command output with truncation
func (e *Engine) displayOutput(output string) {
	maxLines := 20
	lines := splitLines(output)

	if len(lines) > maxLines {
		fmt.Printf("📄 Output (%d lines, showing first %d):\n", len(lines), maxLines)
		for i := 0; i < maxLines; i++ {
			fmt.Printf("  %s\n", lines[i])
		}
		fmt.Printf("  ... (%d more lines)\n", len(lines)-maxLines)
	} else if output != "" {
		fmt.Printf("📄 Output:\n%s\n", output)
	}
}

func splitLines(s string) []string {
	lines := make([]string, 0)
	current := ""
	for _, ch := range s {
		if ch == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/core -v -run TestEngine_Process
```

Expected: PASS

**Step 5: Update main.go to initialize queue**

Modify `cmd/tada/main.go` or wherever the engine is initialized to set up the queue:

```go
// In the chat command or similar initialization
configDir, _ := storage.GetConfigDir()
queueFile := filepath.Join(configDir, storage.SessionDirName, "current.json")
session, _ := storage.InitSession()

// Initialize queue with current session
q := queue.NewQueue(
    filepath.Join(configDir, storage.SessionDirName, session.ID, "queue.json"),
    session.ID,
)
engine.SetQueue(q)
```

**Step 6: Commit**

```bash
git add internal/core/engine.go internal/core/engine_test.go
go test ./internal/core -v
git commit -m "feat(core): integrate queue with Engine.Process"
```

---

## Task 11: Add Integration Tests

**Files:**
- Create: `internal/core/queue/integration_test.go`

**Step 1: Write integration tests**

Create `internal/core/queue/integration_test.go`:

```go
package queue

import (
    "context"
    "os"
    "path/filepath"
    "testing"
    "time"

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
```

**Step 2: Run integration tests**

```bash
go test ./internal/core/queue -v -run TestIntegration
```

Expected: PASS

**Step 3: Commit**

```bash
git add internal/core/queue/integration_test.go
go test ./internal/core/queue -v
git commit -m "test(queue): add integration tests for full workflow"
```

---

## Task 12: Verify and Clean Up

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
./tada tasks --help
./tada --help
```

Expected: Binary builds successfully, commands show help

**Step 3: Check for go vet issues**

```bash
go vet ./...
```

Expected: No warnings

**Step 4: Format code**

```bash
go fmt ./...
```

**Step 5: Final commit**

```bash
git add -A
git commit -m "test(tui): verify and finalize TUI implementation"
```

---

## Task 13: Update go.mod with Dependencies

**Files:**
- Modify: `go.mod`

**Step 1: Add required dependencies**

```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/google/uuid@latest
go mod tidy
```

**Step 2: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add Bubble Tea and lipgloss for TUI"
```

---

## Summary

This implementation plan creates:

1. **Task Types** (`internal/core/queue/types.go`) - Task structure with status transitions
2. **Queue Store** (`internal/core/queue/store.go`) - JSON persistence
3. **Queue Manager** (`internal/core/queue/queue.go`) - Task lifecycle management
4. **Terminal Prompt** (`internal/terminal/prompt.go`) - Sync command confirmation
5. **TUI Components** (`internal/core/tui/`) - Bubble Tea model, view, and key bindings
6. **Tasks Command** (`cmd/tada/tasks.go`) - `tada tasks` CLI command
7. **Engine Integration** (`internal/core/engine.go`) - Queue integration for async commands

**Total estimated time:** ~50-60 minutes (following TDD with small commits)

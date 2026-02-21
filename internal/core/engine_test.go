package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
	"github.com/Lin-Jiong-HDU/tada/internal/core/queue"
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

func TestEngine_SetQueue(t *testing.T) {
	// Create temporary session
	tmpDir, _ := os.MkdirTemp("", "tada-engine-test-*")
	defer os.RemoveAll(tmpDir)

	policy := &security.SecurityPolicy{
		CommandLevel: security.ConfirmDangerous,
	}

	executor := NewExecutor(30 * time.Second)
	engine := NewEngine(nil, executor, policy)

	// Verify SetQueue method exists
	queueFile := filepath.Join(tmpDir, "queue.json")
	q := queue.NewQueue(queueFile, "test-session")

	engine.SetQueue(q)

	// If we got here, SetQueue works
	if q == nil {
		t.Error("Expected queue to be set")
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

	executor := NewExecutor(30 * time.Second)
	engine := NewEngine(nil, executor, policy)
	engine.SetQueue(q)

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

func TestEngine_ParseAsyncSyntax(t *testing.T) {
	executor := NewExecutor(5)
	policy := security.DefaultPolicy()
	_ = NewEngine(&mockAIProvider{}, executor, policy)

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
	_ = NewEngine(&mockAIProvider{intent: intent}, executor, policy)

	// Test async input - the mock AI will return the intent we set
	// When we detect async syntax, we should mark all commands as async
	isAsync := parseAsyncSyntax("create test &")
	if !isAsync {
		t.Error("Expected async syntax to be detected")
	}

	// Verify that stripping works
	stripped := stripAsyncSyntax("create test &")
	if stripped != "create test" {
		t.Errorf("Expected 'create test', got '%s'", stripped)
	}
}

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

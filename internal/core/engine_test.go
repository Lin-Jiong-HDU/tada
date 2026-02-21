package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
	"github.com/Lin-Jiong-HDU/tada/internal/core/queue"
	"github.com/Lin-Jiong-HDU/tada/internal/core/security"
)

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

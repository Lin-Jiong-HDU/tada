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

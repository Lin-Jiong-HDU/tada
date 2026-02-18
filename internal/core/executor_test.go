package core

import (
	"context"
	"testing"
	"time"
)

func TestExecute_SimpleCommand(t *testing.T) {
	executor := NewExecutor(5 * time.Second)

	result, err := executor.Execute(context.Background(), Command{
		Cmd:  "echo",
		Args: []string{"hello", "world"},
	})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Output != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", result.Output)
	}
	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}
}

func TestExecute_CommandNotFound(t *testing.T) {
	executor := NewExecutor(5 * time.Second)

	result, err := executor.Execute(context.Background(), Command{
		Cmd: "nonexistent-command-xyz123",
	})

	// Either an error should be returned, or exit code should be non-zero
	// Platform behavior varies, so we check for either condition
	failed := err != nil || result.Error != nil || result.ExitCode != 0
	if !failed {
		t.Error("Expected some indication of failure for nonexistent command")
	}
}

func TestExecuteBatch_MultipleCommands(t *testing.T) {
	executor := NewExecutor(5 * time.Second)

	commands := []Command{
		{Cmd: "echo", Args: []string{"first"}},
		{Cmd: "echo", Args: []string{"second"}},
	}

	results, err := executor.ExecuteBatch(context.Background(), commands)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
	if results[0].Output != "first" {
		t.Errorf("Expected 'first', got '%s'", results[0].Output)
	}
	if results[1].Output != "second" {
		t.Errorf("Expected 'second', got '%s'", results[1].Output)
	}
}

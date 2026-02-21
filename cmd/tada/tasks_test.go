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

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

func TestRunCommand_HasRunE(t *testing.T) {
	cmd := getRunCommand()
	if cmd.RunE == nil {
		t.Error("Expected RunE function to be set")
	}
}

func TestRunCommand_ShortDesc(t *testing.T) {
	cmd := getRunCommand()
	if cmd.Short == "" {
		t.Error("Expected Short description")
	}
}
